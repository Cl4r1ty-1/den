<script>
  import Header from "../lib/Header.svelte";
  import Modal from "../lib/Modal.svelte";
  import ToastContainer from "../lib/ToastContainer.svelte";

  export let user_count = 0;
  export let node_count = 0;
  export let container_count = 0;

  let nodes = [];
  let users = [];
  let showNodeModal = false;
  let showTokenModal = false;
  let currentToken = "";
  let newNode = {
    name: "",
    hostname: "",
    public_hostname: "",
    max_memory_mb: 4096,
    max_cpu_cores: 4,
    max_storage_gb: 15,
  };
  let toastContainer;
  let activeTab = "nodes";
  let jobs = [];
  let jobsTimer = null;
  let showJobModal = false;
  let jobDetail = null;

  async function loadNodes() {
    const res = await fetch("/admin/nodes");
    const data = await res.json();
    nodes = data.nodes || [];
  }

  async function loadUsers() {
    const res = await fetch("/admin/users");
    const data = await res.json();
    users = data.users || [];
  }

  async function approveUser(userId) {
    const res = await fetch(`/admin/users/${userId}/approve`, {
      method: "POST",
    });
    const data = await res.json();
    if (data.error) {
      toastContainer.addToast(data.error, "danger");
      return;
    }
    toastContainer.addToast("User approved", "success");
    loadUsers();
  }

  let rejectReason = "";
  async function rejectUser(userId) {
    const reason = prompt("Optional reason for rejection:", rejectReason || "");
    rejectReason = reason || "";
    const res = await fetch(`/admin/users/${userId}/reject`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ reason }),
    });
    const data = await res.json();
    if (data.error) {
      toastContainer.addToast(data.error, "danger");
      return;
    }
    toastContainer.addToast("User rejected", "warning");
    loadUsers();
  }

  async function loadJobs() {
    try {
      const res = await fetch("/admin/jobs?limit=50");
      const data = await res.json();
      jobs = data.jobs || [];
    } catch (_) {}
  }

  async function createNode() {
    const res = await fetch("/admin/nodes", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(newNode),
    });
    const data = await res.json();
    if (data.error) {
      toastContainer.addToast(data.error, "danger");
      return;
    }
    showNodeModal = false;
    newNode = {
      name: "",
      hostname: "",
      public_hostname: "",
      max_memory_mb: 4096,
      max_cpu_cores: 4,
      max_storage_gb: 15,
    };
    loadNodes();
    currentToken = data.token;
    showTokenModal = true;
  }

  async function generateToken(nodeId) {
    const res = await fetch(`/admin/nodes/${nodeId}/token`, { method: "GET" });
    const data = await res.json();
    if (data.error) {
      toastContainer.addToast(data.error, "danger");
      return;
    }
    currentToken = data.token;
    showTokenModal = true;
  }

  async function deleteNode(nodeId) {
    if (!confirm("Are you sure you want to delete this node?")) return;

    const res = await fetch(`/admin/nodes/${nodeId}`, { method: "DELETE" });
    const data = await res.json();
    if (data.error) {
      toastContainer.addToast(data.error, "danger");
      return;
    }
    toastContainer.addToast("Node deleted successfully!", "success");
    loadNodes();
  }

  async function deleteUser(userId) {
    if (!confirm("Are you sure you want to delete this user?")) return;

    const res = await fetch(`/admin/users/${userId}`, { method: "DELETE" });
    const data = await res.json();
    if (data.error) {
      toastContainer.addToast(data.error, "danger");
      return;
    }
    toastContainer.addToast("User deleted successfully!", "success");
    loadUsers();
  }

  async function rotateToken(userId) {
    const res = await fetch(`/admin/users/${userId}/rotate-token`, {
      method: "POST",
    });
    const data = await res.json();
    if (data.error) {
      toastContainer.addToast(data.error, "danger");
      return;
    }
    toastContainer.addToast("Container token rotated", "success");
  }

  async function reinstallCLI(userId) {
    const res = await fetch(`/admin/users/${userId}/reinstall-cli`, {
      method: "POST",
    });
    const data = await res.json();
    if (data.error) {
      toastContainer.addToast(data.error, "danger");
      return;
    }
    toastContainer.addToast("CLI reinstalled", "success");
  }

  async function pollJob(jobId) {
    for (let i = 0; i < 90; i++) {
      try {
        const r = await fetch(`/admin/jobs/${jobId}`);
        const j = await r.json();
        if (j && (j.status === "success" || j.status === "failed")) {
          if (j.status === "success") {
            toastContainer.addToast(
              "Container deleted successfully!",
              "success"
            );
            loadUsers();
          } else {
            toastContainer.addToast(j.error || "Delete job failed", "danger");
          }
          return;
        }
      } catch (e) {}
      await new Promise((r) => setTimeout(r, 2000));
    }
    toastContainer.addToast("Timed out waiting for delete job", "danger");
  }

  async function deleteUserContainer(userId) {
    if (!confirm("Are you sure you want to delete this container?")) return;
    const res = await fetch(`/admin/users/${userId}/container`, {
      method: "DELETE",
    });
    const data = await res.json();
    if (data.error) {
      toastContainer.addToast(data.error, "danger");
      return;
    }
    toastContainer.addToast("Deletion queued…", "warning");
    if (data.job_id) {
      pollJob(data.job_id);
    }
  }

  function switchTab(tab) {
    activeTab = tab;
    if (tab === "nodes") {
      loadNodes();
    } else if (tab === "users") {
      loadUsers();
    } else if (tab === "jobs") {
      loadJobs();
      clearInterval(jobsTimer);
      jobsTimer = setInterval(loadJobs, 5000);
    }
  }

  async function openJob(jobId) {
    try {
      const r = await fetch(`/admin/jobs/${jobId}`);
      jobDetail = await r.json();
      showJobModal = true;
    } catch (_) {}
  }

  loadNodes();
</script>

<div class="min-h-screen bg-background text-foreground">
  <Header user={{ is_admin: true }} currentPage="admin" />

  <main class="max-w-6xl mx-auto p-6">
    <div class="mb-8">
      <h1 class="text-4xl font-heading mb-2">admin dashboard</h1>
      <p class="text-foreground/70">manage nodes, users, and containers</p>
    </div>

    <div class="grid md:grid-cols-3 gap-6 mb-8">
      <div
        class="bg-secondary-background border-2 border-border p-6 text-center shadow-shadow"
      >
        <div
          class="w-12 h-12 mx-auto mb-3 bg-chart-2 border-2 border-border flex items-center justify-center"
        >
          <svg
            class="w-6 h-6 text-main-foreground"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M12 4.354a4 4 0 110 5.292M15 21H3v-1a6 6 0 0112 0v1zm0 0h6v-1a6 6 0 00-9-5.197m13.5-9a2.5 2.5 0 11-5 0 2.5 2.5 0 015 0z"
            ></path>
          </svg>
        </div>
        <div class="text-2xl font-heading">{user_count}</div>
        <div class="text-foreground/70 text-sm">users</div>
      </div>

      <div
        class="bg-secondary-background border-2 border-border p-6 text-center shadow-shadow"
      >
        <div
          class="w-12 h-12 mx-auto mb-3 bg-chart-3 border-2 border-border flex items-center justify-center"
        >
          <svg
            class="w-6 h-6 text-main-foreground"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2m-2-4h.01M17 16h.01"
            ></path>
          </svg>
        </div>
        <div class="text-2xl font-heading">{node_count}</div>
        <div class="text-foreground/70 text-sm">nodes</div>
      </div>

      <div
        class="bg-secondary-background border-2 border-border p-6 text-center shadow-shadow"
      >
        <div
          class="w-12 h-12 mx-auto mb-3 bg-chart-1 border-2 border-border flex items-center justify-center"
        >
          <svg
            class="w-6 h-6 text-main-foreground"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10"
            ></path>
          </svg>
        </div>
        <div class="text-2xl font-heading">{container_count}</div>
        <div class="text-foreground/70 text-sm">containers</div>
      </div>
    </div>
    <div class="flex gap-2 mb-6">
      <button
        class="px-4 py-2 border-2 border-border font-heading hover:translate-x-1 hover:translate-y-1 transition-transform {activeTab ===
        'nodes'
          ? 'bg-main text-main-foreground shadow-shadow'
          : 'bg-background text-foreground'}"
        on:click={() => switchTab("nodes")}
      >
        nodes
      </button>
      <button
        class="px-4 py-2 border-2 border-border font-heading hover:translate-x-1 hover:translate-y-1 transition-transform {activeTab ===
        'users'
          ? 'bg-main text-main-foreground shadow-shadow'
          : 'bg-background text-foreground'}"
        on:click={() => switchTab("users")}
      >
        users
      </button>
      <button
        class="px-4 py-2 border-2 border-border font-heading hover:translate-x-1 hover:translate-y-1 transition-transform {activeTab ===
        'jobs'
          ? 'bg-main text-main-foreground shadow-shadow'
          : 'bg-background text-foreground'}"
        on:click={() => switchTab("jobs")}
      >
        jobs
      </button>
    </div>
    {#if activeTab === "nodes"}
      <div
        class="bg-secondary-background border-2 border-border p-6 shadow-shadow"
      >
        <div class="flex items-center justify-between mb-6">
          <h2 class="text-2xl font-heading">node management</h2>
          <button
            class="bg-main text-main-foreground border-2 border-border px-4 py-2 font-heading hover:translate-x-1 hover:translate-y-1 transition-transform shadow-shadow"
            on:click={() => (showNodeModal = true)}
          >
            <svg
              class="w-4 h-4 inline mr-2"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                stroke-linecap="round"
                stroke-linejoin="round"
                stroke-width="2"
                d="M12 6v6m0 0v6m0-6h6m-6 0H6"
              ></path>
            </svg>
            add node
          </button>
        </div>

        {#if nodes.length}
          <div class="grid gap-4">
            {#each nodes as node}
              <div
                class="bg-background border-2 border-border p-4 shadow-shadow"
              >
                <div class="flex items-center justify-between">
                  <div class="flex items-center gap-4">
                    <div
                      class="w-12 h-12 bg-chart-3 border-2 border-border flex items-center justify-center"
                    >
                      <svg
                        class="w-6 h-6 text-main-foreground"
                        fill="none"
                        stroke="currentColor"
                        viewBox="0 0 24 24"
                      >
                        <path
                          stroke-linecap="round"
                          stroke-linejoin="round"
                          stroke-width="2"
                          d="M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2m-2-4h.01M17 16h.01"
                        ></path>
                      </svg>
                    </div>
                    <div>
                      <h3 class="font-heading font-mono">{node.name}</h3>
                      <div class="text-sm text-foreground/70">
                        <div>
                          {node.hostname}
                          {#if node.public_hostname}→ {node.public_hostname}{/if}
                        </div>
                        <div>
                          {node.max_memory_mb}MB / {node.max_cpu_cores} cores / {node.max_storage_gb}GB
                        </div>
                      </div>
                    </div>
                  </div>

                  <div class="flex items-center gap-3">
                    <div class="text-right text-sm">
                      <div
                        class="px-2 py-1 border-2 border-border text-xs font-heading {node.is_online
                          ? 'bg-chart-4 text-main-foreground'
                          : 'bg-chart-1 text-main-foreground'}"
                      >
                        {node.is_online ? "online" : "offline"}
                      </div>
                      <div class="text-foreground/70 mt-1">
                        {node.last_seen
                          ? new Date(node.last_seen).toLocaleString()
                          : "never seen"}
                      </div>
                    </div>

                    <div class="flex gap-2">
                      <button
                        class="bg-chart-2 text-main-foreground border-2 border-border px-3 py-1 text-sm font-heading hover:translate-x-1 hover:translate-y-1 transition-transform shadow-shadow"
                        on:click={() => generateToken(node.id)}
                      >
                        <svg
                          class="w-4 h-4 inline mr-1"
                          fill="none"
                          stroke="currentColor"
                          viewBox="0 0 24 24"
                        >
                          <path
                            stroke-linecap="round"
                            stroke-linejoin="round"
                            stroke-width="2"
                            d="M15 7a2 2 0 012 2m4 0a6 6 0 01-7.743 5.743L11 17H9v2H7v2H4a1 1 0 01-1-1v-2.586a1 1 0 01.293-.707l5.964-5.964A6 6 0 1721 9z"
                          ></path>
                        </svg>
                        new token
                      </button>
                      <button
                        class="bg-chart-1 text-main-foreground border-2 border-border px-3 py-1 text-sm font-heading hover:translate-x-1 hover:translate-y-1 transition-transform shadow-shadow"
                        on:click={() => deleteNode(node.id)}
                      >
                        <svg
                          class="w-4 h-4 inline mr-1"
                          fill="none"
                          stroke="currentColor"
                          viewBox="0 0 24 24"
                        >
                          <path
                            stroke-linecap="round"
                            stroke-linejoin="round"
                            stroke-width="2"
                            d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
                          ></path>
                        </svg>
                        delete
                      </button>
                    </div>
                  </div>
                </div>
              </div>
            {/each}
          </div>
        {:else}
          <div class="text-center py-12">
            <div
              class="w-20 h-20 mx-auto mb-4 bg-foreground/10 border-2 border-border flex items-center justify-center"
            >
              <svg
                class="w-10 h-10 text-foreground/50"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  stroke-width="2"
                  d="M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2m-2-4h.01M17 16h.01"
                ></path>
              </svg>
            </div>
            <h3 class="text-xl font-heading mb-2">no nodes yet</h3>
            <p class="text-foreground/70 mb-6">
              add compute nodes to start hosting containers
            </p>
          </div>
        {/if}
      </div>
    {/if}

    {#if activeTab === "users"}
      <div
        class="bg-secondary-background border-2 border-border p-6 shadow-shadow"
      >
        <div class="flex items-center justify-between mb-6">
          <h2 class="text-2xl font-heading">user management</h2>
        </div>

        {#if users.length}
          <div class="grid gap-4">
            {#each users as user}
              <div
                class="bg-background border-2 border-border p-4 shadow-shadow"
              >
                <div class="flex items-center justify-between">
                  <div class="flex items-center gap-4">
                    <div
                      class="w-12 h-12 bg-chart-2 border-2 border-border flex items-center justify-center"
                    >
                      <svg
                        class="w-6 h-6 text-main-foreground"
                        fill="none"
                        stroke="currentColor"
                        viewBox="0 0 24 24"
                      >
                        <path
                          stroke-linecap="round"
                          stroke-linejoin="round"
                          stroke-width="2"
                          d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z"
                        ></path>
                      </svg>
                    </div>
                    <div>
                      <h3 class="font-heading">{user.display_name}</h3>
                      <div class="text-sm text-foreground/70">
                        <div class="font-mono">@{user.username}</div>
                        <div>{user.email}</div>
                        <div>
                          Joined: {new Date(
                            user.created_at
                          ).toLocaleDateString()}
                        </div>
                      </div>
                    </div>
                  </div>

                  <div class="flex items-center gap-3">
                    <div class="text-right text-sm">
                      {#if user.is_admin}
                        <div
                          class="px-2 py-1 bg-chart-3 text-main-foreground border-2 border-border text-xs font-heading"
                        >
                          admin
                        </div>
                      {/if}
                      {#if user.container_id}
                        <div
                          class="px-2 py-1 bg-chart-4 text-main-foreground border-2 border-border text-xs font-heading mt-1"
                        >
                          has container
                        </div>
                      {/if}
                      {#if user.approval_status}
                        <div
                          class="px-2 py-1 border-2 border-border text-xs font-heading mt-1 {user.approval_status ===
                          'approved'
                            ? 'bg-chart-4 text-main-foreground'
                            : user.approval_status === 'rejected'
                              ? 'bg-chart-1 text-main-foreground'
                              : 'bg-background'}"
                        >
                          {user.approval_status}
                        </div>
                      {/if}
                    </div>

                    <div class="flex gap-2">
                      {#if !user.is_admin}
                        {#if user.approval_status === "pending"}
                          <button
                            class="bg-chart-3 text-main-foreground border-2 border-border px-3 py-1 text-sm font-heading hover:translate-x-1 hover:translate-y-1 transition-transform shadow-shadow"
                            on:click={() => approveUser(user.id)}
                          >
                            approve
                          </button>
                          <button
                            class="bg-chart-1 text-main-foreground border-2 border-border px-3 py-1 text-sm font-heading hover:translate-x-1 hover:translate-y-1 transition-transform shadow-shadow"
                            on:click={() => rejectUser(user.id)}
                          >
                            reject
                          </button>
                        {/if}
                      {/if}
                      {#if user.container_id}
                        <button
                          class="bg-chart-1 text-main-foreground border-2 border-border px-3 py-1 text-sm font-heading hover:translate-x-1 hover:translate-y-1 transition-transform shadow-shadow"
                          on:click={() => deleteUserContainer(user.id)}
                        >
                          <svg
                            class="w-4 h-4 inline mr-1"
                            fill="none"
                            stroke="currentColor"
                            viewBox="0 0 24 24"
                          >
                            <path
                              stroke-linecap="round"
                              stroke-linejoin="round"
                              stroke-width="2"
                              d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
                            ></path>
                          </svg>
                          delete container
                        </button>
                        <button
                          class="bg-chart-2 text-main-foreground border-2 border-border px-3 py-1 text-sm font-heading hover:translate-x-1 hover:translate-y-1 transition-transform shadow-shadow"
                          on:click={() => rotateToken(user.id)}
                        >
                          rotate token
                        </button>
                        <button
                          class="bg-chart-3 text-main-foreground border-2 border-border px-3 py-1 text-sm font-heading hover:translate-x-1 hover:translate-y-1 transition-transform shadow-shadow"
                          on:click={() => reinstallCLI(user.id)}
                        >
                          reinstall cli
                        </button>
                      {/if}
                      <button
                        class="bg-chart-1 text-main-foreground border-2 border-border px-3 py-1 text-sm font-heading hover:translate-x-1 hover:translate-y-1 transition-transform shadow-shadow"
                        on:click={() => deleteUser(user.id)}
                      >
                        <svg
                          class="w-4 h-4 inline mr-1"
                          fill="none"
                          stroke="currentColor"
                          viewBox="0 0 24 24"
                        >
                          <path
                            stroke-linecap="round"
                            stroke-linejoin="round"
                            stroke-width="2"
                            d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
                          ></path>
                        </svg>
                        delete user
                      </button>
                    </div>
                  </div>
                </div>
              </div>
            {/each}
          </div>
        {:else}
          <div class="text-center py-12">
            <div
              class="w-20 h-20 mx-auto mb-4 bg-foreground/10 border-2 border-border flex items-center justify-center"
            >
              <svg
                class="w-10 h-10 text-foreground/50"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  stroke-width="2"
                  d="M12 4.354a4 4 0 110 5.292M15 21H3v-1a6 6 0 0112 0v1zm0 0h6v-1a6 6 0 00-9-5.197m13.5-9a2.5 2.5 0 11-5 0 2.5 2.5 0 015 0z"
                ></path>
              </svg>
            </div>
            <h3 class="text-xl font-heading mb-2">no users yet</h3>
            <p class="text-foreground/70 mb-6">
              users will appear here once they sign up
            </p>
          </div>
        {/if}
      </div>
    {/if}

    {#if activeTab === "jobs"}
      <div
        class="bg-secondary-background border-2 border-border p-6 shadow-shadow"
      >
        <div class="flex items-center justify-between mb-6">
          <h2 class="text-2xl font-heading">recent jobs</h2>
          <button
            class="bg-main text-main-foreground border-2 border-border px-3 py-1 font-heading hover:translate-x-1 hover:translate-y-1 transition-transform shadow-shadow"
            on:click={loadJobs}
          >
            refresh
          </button>
        </div>
        {#if jobs.length}
          <div class="overflow-x-auto">
            <table class="w-full text-sm">
              <thead>
                <tr class="text-left">
                  <th
                    class="border-2 border-border bg-background p-2 font-heading"
                    >id</th
                  >
                  <th
                    class="border-2 border-border bg-background p-2 font-heading"
                    >type</th
                  >
                  <th
                    class="border-2 border-border bg-background p-2 font-heading"
                    >status</th
                  >
                  <th
                    class="border-2 border-border bg-background p-2 font-heading"
                    >error</th
                  >
                  <th
                    class="border-2 border-border bg-background p-2 font-heading"
                    >updated</th
                  >
                </tr>
              </thead>
              <tbody>
                {#each jobs as j}
                  <tr
                    class="cursor-pointer hover:bg-foreground/5"
                    on:click={() => openJob(j.id)}
                  >
                    <td class="border-2 border-border p-2 font-mono">{j.id}</td>
                    <td class="border-2 border-border p-2">{j.type}</td>
                    <td class="border-2 border-border p-2">
                      <span
                        class="px-2 py-1 border-2 border-border text-xs font-heading {j.status ===
                        'success'
                          ? 'bg-chart-4 text-main-foreground'
                          : j.status === 'failed'
                            ? 'bg-chart-1 text-main-foreground'
                            : 'bg-background'}"
                      >
                        {j.status}
                      </span>
                    </td>
                    <td class="border-2 border-border p-2 text-foreground/70"
                      >{j.error || ""}</td
                    >
                    <td class="border-2 border-border p-2"
                      >{new Date(j.updated_at).toLocaleString()}</td
                    >
                  </tr>
                {/each}
              </tbody>
            </table>
          </div>
        {:else}
          <p class="text-foreground/70">no jobs yet</p>
        {/if}
      </div>
    {/if}
  </main>
</div>

<Modal
  show={showNodeModal}
  title="Add Node"
  onClose={() => (showNodeModal = false)}
>
  <form on:submit|preventDefault={createNode} class="space-y-4">
    <div class="grid md:grid-cols-2 gap-4">
      <div>
        <label class="block text-sm font-heading mb-2" for="node_name"
          >node name</label
        >
        <input
          id="node_name"
          type="text"
          bind:value={newNode.name}
          required
          class="w-full bg-background border-2 border-border p-3"
          placeholder="node-1"
        />
      </div>
      <div>
        <label class="block text-sm font-heading mb-2" for="node_hostname"
          >internal hostname</label
        >
        <input
          id="node_hostname"
          type="text"
          bind:value={newNode.hostname}
          required
          class="w-full bg-background border-2 border-border p-3"
          placeholder="192.168.1.100"
        />
      </div>
    </div>

    <div>
      <label class="block text-sm font-heading mb-2" for="node_public_hostname"
        >public hostname (optional)</label
      >
      <input
        id="node_public_hostname"
        type="text"
        bind:value={newNode.public_hostname}
        class="w-full bg-background border-2 border-border p-3"
        placeholder="node1.den.dev"
      />
    </div>

    <div class="grid grid-cols-3 gap-4">
      <div>
        <label class="block text-sm font-heading mb-2" for="node_mem"
          >memory (MB)</label
        >
        <input
          id="node_mem"
          type="number"
          bind:value={newNode.max_memory_mb}
          class="w-full bg-background border-2 border-border p-3"
        />
      </div>
      <div>
        <label class="block text-sm font-heading mb-2" for="node_cores"
          >cpu cores</label
        >
        <input
          id="node_cores"
          type="number"
          bind:value={newNode.max_cpu_cores}
          class="w-full bg-background border-2 border-border p-3"
        />
      </div>
      <div>
        <label class="block text-sm font-heading mb-2" for="node_storage"
          >storage (GB)</label
        >
        <input
          id="node_storage"
          type="number"
          bind:value={newNode.max_storage_gb}
          class="w-full bg-background border-2 border-border p-3"
        />
      </div>
    </div>
  </form>

  <div slot="footer" class="flex gap-3">
    <button
      class="bg-foreground/10 border-2 border-border px-4 py-2 font-heading hover:translate-x-1 hover:translate-y-1 transition-transform"
      on:click={() => (showNodeModal = false)}
    >
      cancel
    </button>
    <button
      class="bg-main text-main-foreground border-2 border-border px-4 py-2 font-heading hover:translate-x-1 hover:translate-y-1 transition-transform shadow-shadow"
      on:click={createNode}
    >
      create node
    </button>
  </div>
</Modal>

<Modal
  show={showTokenModal}
  title="Node Token"
  onClose={() => (showTokenModal = false)}
>
  <div class="space-y-4">
    <p class="text-foreground/70">
      Copy this token and use it to authenticate the node:
    </p>
    <div
      class="bg-background border-2 border-border p-4 font-mono text-sm break-all"
    >
      {currentToken}
    </div>
    <p class="text-sm text-foreground/70">
      <strong>Important:</strong> Save this token securely. It won't be shown again.
    </p>
  </div>

  <div slot="footer" class="flex gap-3">
    <button
      class="bg-main text-main-foreground border-2 border-border px-4 py-2 font-heading hover:translate-x-1 hover:translate-y-1 transition-transform shadow-shadow"
      on:click={() => {
        navigator.clipboard.writeText(currentToken);
        toastContainer.addToast("Token copied!", "success");
      }}
    >
      copy token
    </button>
    <button
      class="bg-foreground/10 border-2 border-border px-4 py-2 font-heading hover:translate-x-1 hover:translate-y-1 transition-transform"
      on:click={() => (showTokenModal = false)}
    >
      close
    </button>
  </div>
</Modal>

<Modal
  show={showJobModal}
  title="Job Details"
  onClose={() => (showJobModal = false)}
>
  {#if jobDetail}
    <div class="space-y-3">
      <div>
        <span class="font-heading">id:</span>
        <span class="font-mono">{jobDetail.id}</span>
      </div>
      <div><span class="font-heading">type:</span> {jobDetail.type}</div>
      <div><span class="font-heading">status:</span> {jobDetail.status}</div>
      {#if jobDetail.error}
        <div class="text-chart-1">
          <span class="font-heading">error:</span>
          <span class="font-mono break-all">{jobDetail.error}</span>
        </div>
      {/if}
      {#if jobDetail.result}
        <div>
          <span class="font-heading">result:</span>
          <pre
            class="bg-background border-2 border-border p-2 overflow-auto text-xs">{jobDetail.result}</pre>
        </div>
      {/if}
      <div class="text-sm text-foreground/70">
        created: {new Date(jobDetail.created_at).toLocaleString()} | updated: {new Date(
          jobDetail.updated_at
        ).toLocaleString()}
      </div>
    </div>
  {:else}
    <p>Loading…</p>
  {/if}
</Modal>

<ToastContainer bind:this={toastContainer} />
