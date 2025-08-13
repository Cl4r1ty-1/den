import React from 'react'
import { createRoot } from 'react-dom/client'
import { BrowserRouter, Routes, Route, Navigate, Link } from 'react-router-dom'
import gsap from 'gsap'
import './styles.css'

function Layout({ children }: React.PropsWithChildren) {
  const navRef = React.useRef<HTMLDivElement>(null)
  React.useEffect(() => {
    if (!navRef.current) return
    gsap.fromTo(navRef.current.querySelectorAll('a'), { y: -10, opacity: 0 }, { y:0, opacity:1, stagger:.06, duration:.6, ease:'power3.out' })
  }, [])
  return (
    <div className="container">
      <div className="glass nav" ref={navRef}>
        <div className="title">den</div>
        <Link to="/">home</Link>
        <Link to="/dashboard">dashboard</Link>
        <Link to="/ssh">ssh</Link>
        <Link to="/aup">aup</Link>
        <Link to="/admin" style={{marginLeft:'auto'}}>admin</Link>
        <a href="/login">login</a>
      </div>
      {children}
    </div>
  )
}

function Home() {
  const hero = React.useRef<HTMLDivElement>(null)
  React.useEffect(()=>{ if(hero.current){ gsap.fromTo(hero.current, { y: 16, opacity:0 }, { y: 0, opacity:1, duration:.8, ease:'power3.out' })}},[])
  return (
    <div ref={hero} className="glass card" style={{ marginTop:'1rem' }}>
      <div style={{ fontSize:'1.25rem', fontWeight:700, marginBottom:'.25rem' }}>hi, welcome to den</div>
      <div className="muted">a cozy pubnix for tinkering and projects</div>
    </div>
  )
}

function Dashboard() {
  const [container, setContainer] = React.useState<any>(null)
  const [subs, setSubs] = React.useState<any[]>([])
  const [sd, setSd] = React.useState({ subdomain:'', target_port:'', subdomain_type:'project' })
  const [busy, setBusy] = React.useState(false)
  const load = React.useCallback(()=>{
    fetch('/user/container').then(r=>r.json()).then(setContainer).catch(()=>{})
    fetch('/user/api/subdomains').then(r=>r.json()).then(d=>setSubs(d.subdomains||[])).catch(()=>{})
  }, [])
  React.useEffect(()=>{ load() }, [load])
  const newPort = async () => { await fetch('/user/container/ports/new', { method:'POST' }); load() }
  const create = async () => {
    setBusy(true)
    const payload = { ...sd, target_port: parseInt(sd.target_port) }
    const r = await fetch('/user/subdomains', { method:'POST', headers:{'Content-Type':'application/json'}, body: JSON.stringify(payload) })
    const d = await r.json(); setBusy(false)
    if (d.error) { alert(d.error); return }
    setSd({ subdomain:'', target_port:'', subdomain_type:'project' }); load()
  }
  const del = async (id:number) => { const d = await fetch(`/user/subdomains/${id}`, { method:'DELETE' }).then(r=>r.json()); if (d.error) alert(d.error); else load() }
  return (
    <div className="grid two" style={{ marginTop:'1rem' }}>
      <div className="card">
        <div style={{ display:'flex', justifyContent:'space-between', alignItems:'center' }}>
          <div style={{ fontWeight:700 }}>container</div>
          <button className="btn" onClick={newPort}>get new port</button>
        </div>
        <pre style={{background:'transparent', border:'1px solid rgba(255,255,255,.06)', borderRadius:8, padding:'.5rem', marginTop:'.5rem'}}>{JSON.stringify(container, null, 2)}</pre>
      </div>
      <div className="card">
        <div style={{ fontWeight:700 }}>subdomains</div>
        <table style={{ marginTop:'.5rem' }}><thead><tr><th>subdomain</th><th>port</th><th>type</th><th></th></tr></thead>
        <tbody>
          {subs.map(s => <tr key={s.id}><td>{s.subdomain}</td><td>{s.target_port}</td><td>{s.subdomain_type}</td><td><button className="btn secondary" onClick={()=>del(s.id)}>delete</button></td></tr>)}
        </tbody></table>
        <div style={{marginTop:'.5rem', display:'flex', gap:'.5rem', flexWrap:'wrap'}}>
          <select value={sd.subdomain_type} onChange={e=>setSd(v=>({ ...v, subdomain_type:(e.target as HTMLSelectElement).value }))}>
            <option value="project">project</option>
            <option value="username">username</option>
          </select>
          <input placeholder="subdomain" value={sd.subdomain} onChange={e=>setSd(v=>({ ...v, subdomain:(e.target as HTMLInputElement).value }))} />
          <input placeholder="port" value={sd.target_port} onChange={e=>setSd(v=>({ ...v, target_port:(e.target as HTMLInputElement).value }))} style={{ width:120 }} />
          <button className="btn" disabled={busy} onClick={create}>create</button>
        </div>
      </div>
    </div>
  )
}

function AUP() {
  const [qs, setQs] = React.useState<{id:number; prompt:string}[]>([])
  const [answers, setAnswers] = React.useState<Record<number, string>>({})
  const [agreeT, setAT] = React.useState(false)
  const [agreeP, setAP] = React.useState(false)
  const [err, setErr] = React.useState('')
  React.useEffect(()=>{ fetch('/user/aup/questions').then(r=>r.json()).then(d=>setQs(d.questions||[])) }, [])
  const submit = async () => {
    setErr('')
    const payload = { accept_tos: agreeT, accept_privacy: agreeP, answers: qs.map(q => ({ id:q.id, answer: answers[q.id]||'' })) }
    const r = await fetch('/user/aup/accept', { method:'POST', headers:{'Content-Type':'application/json'}, body: JSON.stringify(payload) })
    const d = await r.json(); if (d.error) { setErr(d.error); return } ; window.location.href='/user/dashboard'
  }
  return (
    <div className="card" style={{ marginTop:'1rem' }}>
      <div style={{ fontWeight:700, marginBottom:'.5rem' }}>aup & privacy</div>
      <label><input type="checkbox" checked={agreeT} onChange={e=>setAT(e.target.checked)} /> <span className="muted">i agree to the acceptable use policy</span></label><br/>
      <label><input type="checkbox" checked={agreeP} onChange={e=>setAP(e.target.checked)} /> <span className="muted">i agree to the privacy policy</span></label>
      <div style={{ fontWeight:600, marginTop:'.75rem' }}>quiz</div>
      {qs.map(q => <div key={q.id} style={{margin:'0.5rem 0'}}><div style={{fontWeight:600}}>{q.prompt}</div><input className="glass" style={{padding:'0.5rem', width:'100%', maxWidth:520, border:'1px solid rgba(255,255,255,.06)'}} value={answers[q.id]||''} onChange={e=>setAnswers(a=>({ ...a, [q.id]:(e.target as HTMLInputElement).value }))} /></div>)}
      {err && <div style={{color:'#f87171', marginTop:'.5rem'}}>{err}</div>}
      <div style={{marginTop:'1rem'}}><button className="btn" onClick={submit}>submit</button></div>
    </div>
  )
}

function SSH() {
  const [method, setMethod] = React.useState<'password'|'key'>('password')
  const [password, setPassword] = React.useState('')
  const [pub, setPub] = React.useState('')
  const submit = async () => {
    const body = method==='password' ? { method, password } : { method:'key', public_key: pub }
    const r = await fetch('/user/ssh-setup', { method:'POST', headers:{'Content-Type':'application/json'}, body: JSON.stringify(body) })
    const d = await r.json(); if (d.error) alert(d.error); else alert('updated')
  }
  return (
    <div className="card" style={{ marginTop:'1rem' }}>
      <div style={{ fontWeight:700, marginBottom:'.5rem' }}>ssh setup</div>
      <label><input type="radio" checked={method==='password'} onChange={()=>setMethod('password')} /> <span className="muted">password</span></label>
      <label style={{marginLeft:'1rem'}}><input type="radio" checked={method==='key'} onChange={()=>setMethod('key')} /> <span className="muted">public key</span></label>
      {method==='password' ? (<div style={{marginTop:'.5rem'}}><input className="glass" type="password" value={password} onChange={e=>setPassword(e.target.value)} placeholder="new password" /></div>) :
        (<div style={{marginTop:'.5rem'}}><textarea className="glass" value={pub} onChange={e=>setPub(e.target.value)} placeholder="ssh-ed25519 ..." rows={5} cols={60} /></div>)}
      <div style={{marginTop:'.5rem'}}><button className="btn" onClick={submit}>save</button></div>
    </div>
  )
}

function Admin() {
  const [nodes, setNodes] = React.useState<any[]>([])
  const [users, setUsers] = React.useState<any[]>([])
  const [form, setForm] = React.useState({ name:'', hostname:'', public_hostname:'', max_memory_mb:4096, max_cpu_cores:4, max_storage_gb:15 })
  const load = React.useCallback(()=>{
    fetch('/admin/nodes').then(r=>r.json()).then(d=>setNodes(d.nodes||[]))
    fetch('/admin/users').then(r=>r.json()).then(d=>setUsers(d.users||[]))
  }, [])
  React.useEffect(()=>{ load() }, [load])
  const addNode = async () => {
    const payload:any = { ...form }
    if (!payload.public_hostname) delete payload.public_hostname
    const d = await fetch('/admin/nodes', { method:'POST', headers:{'Content-Type':'application/json'}, body: JSON.stringify(payload) }).then(r=>r.json())
    if (d.error) alert(d.error); else { alert('node created. token: '+d.token); setForm({ name:'', hostname:'', public_hostname:'', max_memory_mb:4096, max_cpu_cores:4, max_storage_gb:15 }); load() }
  }
  const delNode = async (id:number) => { const d=await fetch(`/admin/nodes/${id}`,{method:'DELETE'}).then(r=>r.json()); if(d.error) alert(d.error); else load() }
  const newToken = async (id:number) => { const d=await fetch(`/admin/nodes/${id}/token`).then(r=>r.json()); if(d.error) alert(d.error); else alert('new token: '+d.token) }
  const delUser = async (id:number) => { const d=await fetch(`/admin/users/${id}`,{method:'DELETE'}).then(r=>r.json()); if(d.error) alert(d.error); else load() }
  const delUserContainer = async (id:number) => { const d=await fetch(`/admin/users/${id}/container`,{method:'DELETE'}).then(r=>r.json()); if(d.error) alert(d.error); else load() }
  const exportUser = async (id:number) => { const ttl=parseInt(prompt('TTL days','7')||'7'); const d=await fetch(`/admin/users/${id}/export`,{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({ttl_days:ttl})}).then(r=>r.json()); if(d.error) alert(d.error); else if(d.download_url) window.open(d.download_url,'_blank'); else alert('export started') }
  return (
    <div>
      <h2>admin</h2>
      <h3>nodes</h3>
      <table><thead><tr><th>name</th><th>hostname</th><th>public</th><th>res</th><th>status</th><th></th></tr></thead>
      <tbody>
        {nodes.map(n => (
          <tr key={n.id}><td>{n.name}</td><td>{n.hostname}</td><td>{n.public_hostname||''}</td><td>{n.max_memory_mb}MB/{n.max_cpu_cores}c/{n.max_storage_gb}GB</td><td>{n.is_online?'online':'offline'}</td>
          <td><button onClick={()=>newToken(n.id)}>new token</button> <button onClick={()=>delNode(n.id)}>delete</button></td></tr>
        ))}
      </tbody></table>
      <div style={{marginTop:'.5rem'}}>
        <input placeholder="name" value={form.name} onChange={e=>setForm(v=>({...v, name:e.target.value}))} />
        <input placeholder="hostname" value={form.hostname} onChange={e=>setForm(v=>({...v, hostname:e.target.value}))} style={{marginLeft:'.5rem'}} />
        <input placeholder="public hostname (opt)" value={form.public_hostname} onChange={e=>setForm(v=>({...v, public_hostname:e.target.value}))} style={{marginLeft:'.5rem'}} />
        <button onClick={addNode} style={{marginLeft:'.5rem'}}>add node</button>
      </div>
      <h3 style={{marginTop:'1rem'}}>users</h3>
      <table><thead><tr><th>username</th><th>display</th><th>email</th><th>container</th><th>admin</th><th>created</th><th></th></tr></thead>
      <tbody>
        {users.map(u => (
          <tr key={u.id}><td>{u.username}</td><td>{u.display_name}</td><td>{u.email}</td><td>{u.container_id||'none'}</td><td>{u.is_admin?'yes':'no'}</td><td>{new Date(u.created_at).toLocaleDateString()}</td>
          <td><button disabled={!u.container_id} onClick={()=>delUserContainer(u.id)}>delete container</button> <button disabled={!u.container_id} onClick={()=>exportUser(u.id)}>download data</button> <button onClick={()=>delUser(u.id)} style={{color:'crimson'}}>delete user</button></td></tr>
        ))}
      </tbody></table>
    </div>
  )
}

function App() {
  React.useEffect(()=>{ fetch('/user/me').then(r=>r.json()).then(u => { (window as any).__denUser=u }) }, [])
  return (
    <BrowserRouter basename="/app">
      <Layout>
        <Routes>
          <Route path="/" element={<Home/>} />
          <Route path="/dashboard" element={<Dashboard/>} />
          <Route path="/ssh" element={<SSH/>} />
          <Route path="/aup" element={<AUP/>} />
          <Route path="/admin" element={<Admin/>} />
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </Layout>
    </BrowserRouter>
  )
}

createRoot(document.getElementById('root')!).render(<App />)

