package ssh

import (
	"bytes"
	"database/sql"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"

	"github.com/den/internal/database"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/crypto/ssh"
)

type Gateway struct {
	db       *database.DB
	hostKey  ssh.Signer
	listener net.Listener
}

func NewGateway(db *database.DB) *Gateway {
	return &Gateway{
		db: db,
	}
}

func (g *Gateway) Start() error {
	if err := g.loadHostKey(); err != nil {
		return fmt.Errorf("failed to load host key: %w", err)
	}
		config := &ssh.ServerConfig{
		PublicKeyCallback: g.authenticateUser,
		PasswordCallback:  g.authenticatePassword,
	}
	config.AddHostKey(g.hostKey)

	listener, err := net.Listen("tcp", ":22")
	if err != nil {
		return fmt.Errorf("failed to listen on SSH port: %w", err)
	}
	g.listener = listener

	log.Println("ssh Gateway listening on :22")

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("failed to accept ssh connection: %v", err)
			continue
		}

		go g.handleConnection(conn, config)
	}
}

func (g *Gateway) Stop() error {
	if g.listener != nil {
		return g.listener.Close()
	}
	return nil
}

func (g *Gateway) loadHostKey() error {
	keyPath := "/etc/ssh/ssh_host_rsa_key"
	keyBytes, err := os.ReadFile(keyPath)
	if err == nil {
		key, err := ssh.ParsePrivateKey(keyBytes)
		if err == nil {
			g.hostKey = key
			return nil
		}
	}
	cmd := exec.Command("ssh-keygen", "-t", "rsa", "-b", "2048", "-f", keyPath, "-N", "")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to generate host key: %w", err)
	}
	keyBytes, err = os.ReadFile(keyPath)
	if err != nil {
		return fmt.Errorf("failed to read generated key: %w", err)
	}

	key, err := ssh.ParsePrivateKey(keyBytes)
	if err != nil {
		return fmt.Errorf("failed to parse generated key: %w", err)
	}

	g.hostKey = key
	return nil
}

func (g *Gateway) authenticateUser(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
	username := conn.User()
	var userID int
	var containerID sql.NullString
	var nodeHostname sql.NullString
	var storedKey sql.NullString

	err := g.db.QueryRow(`
		SELECT u.id, u.container_id, u.ssh_public_key, n.hostname
		FROM users u
		LEFT JOIN containers c ON u.container_id = c.id
		LEFT JOIN nodes n ON c.node_id = n.id
		WHERE u.username = $1
	`, username).Scan(&userID, &containerID, &storedKey, &nodeHostname)

	if err != nil {
		log.Printf("User %s not found: %v", username, err)
		return nil, fmt.Errorf("user not found")
	}

	if !storedKey.Valid {
		return nil, fmt.Errorf("no public key configured")
	}
	storedPublicKey, _, _, _, err := ssh.ParseAuthorizedKey([]byte(storedKey.String))
	if err != nil {
		log.Printf("Failed to parse stored key for %s: %v", username, err)
		return nil, fmt.Errorf("invalid stored key")
	}
	if !bytes.Equal(key.Marshal(), storedPublicKey.Marshal()) {
		return nil, fmt.Errorf("key mismatch")
	}
	permissions := &ssh.Permissions{
		Extensions: map[string]string{
			"user_id":       fmt.Sprintf("%d", userID),
			"username":      username,
			"container_id":  containerID.String,
			"node_hostname": nodeHostname.String,
		},
	}

	return permissions, nil
}

func (g *Gateway) authenticatePassword(conn ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {
	username := conn.User()
	log.Printf("ssh password auth attempt for user: %s", username)
	
	var userID int
	var containerID sql.NullString
	var nodeHostname sql.NullString
	var hashedPassword sql.NullString

	err := g.db.QueryRow(`
		SELECT u.id, u.container_id, u.ssh_password, n.hostname
		FROM users u
		LEFT JOIN containers c ON u.container_id = c.id
		LEFT JOIN nodes n ON c.node_id = n.id
		WHERE u.username = $1
	`, username).Scan(&userID, &containerID, &hashedPassword, &nodeHostname)

	if err != nil {
		log.Printf("database lookup failed for user %s: %v", username, err)
		return nil, fmt.Errorf("user not found")
	}

	log.Printf("user %s found - ID: %d, container: %s, Node: %s, hasPassword: %v", 
		username, userID, containerID.String, nodeHostname.String, hashedPassword.Valid)

	if !hashedPassword.Valid {
		log.Printf("no password configured for user %s", username)
		return nil, fmt.Errorf("no password configured")
	}

	// imagine having security lol :3
	err = bcrypt.CompareHashAndPassword([]byte(hashedPassword.String), password)
	if err != nil {
		log.Printf("password mismatch for user %s", username)
		return nil, fmt.Errorf("password mismatch")
	}
	
	log.Printf("password authentication successful for user %s", username)
	permissions := &ssh.Permissions{
		Extensions: map[string]string{
			"user_id":       fmt.Sprintf("%d", userID),
			"username":      username,
			"container_id":  containerID.String,
			"node_hostname": nodeHostname.String,
		},
	}

	return permissions, nil
}

func (g *Gateway) handleConnection(conn net.Conn, config *ssh.ServerConfig) {
	defer conn.Close()
	sshConn, chans, reqs, err := ssh.NewServerConn(conn, config)
	if err != nil {
		log.Printf("Failed to handshake: %v", err)
		return
	}
	defer sshConn.Close()
	permissions := sshConn.Permissions
	if permissions == nil {
		log.Println("No permissions found")
		return
	}

	containerID := permissions.Extensions["container_id"]
	nodeHostname := permissions.Extensions["node_hostname"]
	username := permissions.Extensions["username"]

	if containerID == "" {
		g.handleNoContainer(sshConn, chans, reqs, username)
		return
	}

	if nodeHostname == "" {
		log.Printf("No node hostname for container %s", containerID)
		return
	}
	g.routeToNode(sshConn, chans, reqs, nodeHostname, containerID, username)
}

func (g *Gateway) handleNoContainer(sshConn *ssh.ServerConn, chans <-chan ssh.NewChannel, reqs <-chan *ssh.Request, username string) {
	go ssh.DiscardRequests(reqs)

	for newChannel := range chans {
		if newChannel.ChannelType() != "session" {
			newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")
			continue
		}

		channel, _, err := newChannel.Accept()
		if err != nil {
			log.Printf("could not accept channel: %v", err)
			continue
		}
		message := fmt.Sprintf("hey there %s!\n\nyou don't have an account yet.\nplease visit https://hack.kim/user/dashboard to create one.\n\n", username)
		channel.Write([]byte(message))
		channel.Close()
	}
}

func (g *Gateway) routeToNode(sshConn *ssh.ServerConn, chans <-chan ssh.NewChannel, reqs <-chan *ssh.Request, nodeHostname, containerID, username string) {
	log.Printf("Routing SSH connection for user %s to container %s on node %s", username, containerID, nodeHostname)
	masterKeyBytes, err := os.ReadFile("/root/.ssh/den_master_key")
	if err != nil {
		log.Printf("failed to read master key: %v", err)
		return
	}
	
	masterKey, err := ssh.ParsePrivateKey(masterKeyBytes)
	if err != nil {
		log.Printf("failed to parse master key: %v", err)
		return
	}
	nodeConn, err := ssh.Dial("tcp", nodeHostname+":22", &ssh.ClientConfig{
		User:            "root",
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(masterKey)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	})
	if err != nil {
		log.Printf("failed to connect to node %s: %v", nodeHostname, err)
		return
	}
	defer nodeConn.Close()
	go func() {
		for req := range reqs {
			if req.Type == "keepalive@openssh.com" {
				if req.WantReply {
					req.Reply(true, nil)
				}
				continue
			}
			if req.WantReply {
				req.Reply(true, nil)
			}
		}
	}()
	for newChannel := range chans {
		if newChannel.ChannelType() != "session" {
			newChannel.Reject(ssh.UnknownChannelType, "unsupported channel type")
			continue
		}

		channel, channelReqs, err := newChannel.Accept()
		if err != nil {
			log.Printf("failed to accept channel: %v", err)
			continue
		}

		go g.handleLXCSession(nodeConn, channel, channelReqs, containerID, username)
	}
}

func (g *Gateway) handleLXCSession(nodeConn *ssh.Client, channel ssh.Channel, reqs <-chan *ssh.Request, containerID, username string) {
	defer channel.Close()
	
	session, err := nodeConn.NewSession()
	if err != nil {
		log.Printf("failed to create session on node: %v", err)
		return
	}
	defer session.Close()
	session.Stdout = channel
	session.Stderr = channel
	session.Stdin = channel

	var shellStarted bool

	for req := range reqs {
		switch req.Type {
		case "pty-req":
			err := session.RequestPty("xterm", 80, 24, ssh.TerminalModes{})
			if req.WantReply {
				req.Reply(err == nil, nil)
			}
		case "shell":
			if !shellStarted {
				shellStarted = true
				cmd := fmt.Sprintf("lxc exec %s -- sudo -u %s -i", containerID, username)
				log.Printf("starting container shell: %s", cmd)
				
				go func() {
					err := session.Run(cmd)
					if err != nil {
						log.Printf("container shell execution failed: %v", err)
					}
				}()
				
				if req.WantReply {
					req.Reply(true, nil)
				}
			} else {
				if req.WantReply {
					req.Reply(false, nil)
				}
			}
		case "window-change":
			if req.WantReply {
				req.Reply(true, nil)
			}
		default:
			if req.WantReply {
				req.Reply(false, nil)
			}
		}
	}
}

func (g *Gateway) forwardRequests(from <-chan *ssh.Request, to ssh.Channel) {
	for req := range from {
		ok, err := to.SendRequest(req.Type, req.WantReply, req.Payload)
		if req.WantReply {
			req.Reply(ok, nil)
		}
		if err != nil {
			return
		}
	}
}

func (g *Gateway) forwardData(from ssh.Channel, to ssh.Channel) {
	defer from.Close()
	defer to.Close()
	go func() {
		defer to.CloseWrite()
		io.Copy(to, from)
	}()
	
	defer from.CloseWrite()
	io.Copy(from, to)
}

func (g *Gateway) handleContainerSession(nodeConn *ssh.Client, channel ssh.Channel, reqs <-chan *ssh.Request, containerID, username string) {
	defer channel.Close()
	session, err := nodeConn.NewSession()
	if err != nil {
		log.Printf("failed to create session on node: %v", err)
		return
	}
	defer session.Close()

	session.Stdout = channel
	session.Stderr = channel
	session.Stdin = channel

	go func() {
		for req := range reqs {
			switch req.Type {
			case "pty-req":
				session.RequestPty("xterm", 80, 24, ssh.TerminalModes{})
				if req.WantReply {
					req.Reply(true, nil)
				}
			case "shell":
				cmd := fmt.Sprintf("lxc exec %s -- bash", containerID)
				go func() {
					err := session.Start(cmd)
					if err != nil {
						log.Printf("container shell start failed: %v", err)
					}
				}()
				if req.WantReply {
					req.Reply(true, nil)
				}
			case "exec":
				if req.WantReply {
					req.Reply(true, nil)
				}
			default:
				if req.WantReply {
					req.Reply(false, nil)
				}
			}
		}
	}()
	session.Wait()
}
