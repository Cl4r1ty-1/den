<script>
	import Header from '../lib/Header.svelte'
	import Modal from '../lib/Modal.svelte'
	import ToastContainer from '../lib/ToastContainer.svelte'
	
	export let user_count = 0
	export let node_count = 0
	export let container_count = 0

	let nodes = []
	let users = []
	let showNodeModal = false
	let showTokenModal = false
	let currentToken = ''
	let newNode = { name: '', hostname: '', public_hostname: '', max_memory_mb: 4096, max_cpu_cores: 4, max_storage_gb: 15 }
	let toastContainer
	let activeTab = 'nodes'

	async function loadNodes() {
		const res = await fetch('/admin/nodes')
		const data = await res.json()
		nodes = data.nodes || []
	}

	async function loadUsers() {
		const res = await fetch('/admin/users')
		const data = await res.json()
		users = data.users || []
	}

	async function createNode() {
		const res = await fetch('/admin/nodes', {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify(newNode)
		})
		const data = await res.json()
		if (data.error) {
			toastContainer.addToast(data.error, 'danger')
			return
		}
		showNodeModal = false
		newNode = { name: '', hostname: '', public_hostname: '', max_memory_mb: 4096, max_cpu_cores: 4, max_storage_gb: 15 }
		loadNodes()
		currentToken = data.token
		showTokenModal = true
	}

	async function generateToken(nodeId) {
		const res = await fetch(`/admin/nodes/${nodeId}/token`, { method: 'GET' })
		const data = await res.json()
		if (data.error) {
			toastContainer.addToast(data.error, 'danger')
			return
		}
		currentToken = data.token
		showTokenModal = true
	}

	async function deleteNode(nodeId) {
		const res = await fetch(`/admin/nodes/${nodeId}`, { method: 'DELETE' })
		const data = await res.json()
		if (data.error) {
			toastContainer.addToast(data.error, 'danger')
			return
		}
		toastContainer.addToast('Node deleted successfully', 'success')
		loadNodes()
	}

	async function deleteUser(userId) {
		const res = await fetch(`/admin/users/${userId}`, { method: 'DELETE' })
		const data = await res.json()
		if (data.error) {
			toastContainer.addToast(data.error, 'danger')
			return
		}
		toastContainer.addToast('User deleted successfully', 'success')
		loadUsers()
	}

	async function deleteUserContainer(userId) {
		const res = await fetch(`/admin/users/${userId}/container`, { method: 'DELETE' })
		const data = await res.json()
		if (data.error) {
			toastContainer.addToast(data.error, 'danger')
			return
		}
		toastContainer.addToast('Container deleted successfully', 'success')
		loadUsers()
	}
	
	function copyToken() {
		navigator.clipboard.writeText(currentToken)
		toastContainer.addToast('Token copied to clipboard!', 'success')
	}

	loadNodes()
	loadUsers()
</script>

<Header user={{ IsAdmin: true }} currentPage="admin" />

<main class="nb-container py-8">
	<div class="mb-8">
		<h1 class="nb-title text-4xl mb-2">
			<span class="text-[var(--nb-warning)]">admin</span> dashboard
		</h1>
		<p class="nb-subtitle text-xl">manage your den infrastructure</p>
	</div>

	<div class="grid md:grid-cols-3 gap-6 mb-8">
		<div class="nb-card-lg text-center">
			<div class="w-16 h-16 mx-auto mb-4 bg-[var(--nb-primary)] rounded-full flex items-center justify-center">
				<svg class="w-8 h-8 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4.354a4 4 0 110 5.292M15 21H3v-1a6 6 0 0112 0v1zm0 0h6v-1a6 6 0 00-9-5.197m13.5-9a2.5 2.5 0 11-5 0 2.5 2.5 0 015 0z"></path>
				</svg>
			</div>
			<div class="nb-title text-3xl mb-1">{user_count}</div>
			<div class="nb-text-muted">total users</div>
		</div>
		
		<div class="nb-card-lg text-center">
			<div class="w-16 h-16 mx-auto mb-4 bg-[var(--nb-secondary)] rounded-full flex items-center justify-center">
				<svg class="w-8 h-8 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2m-2-4h.01M17 16h.01"></path>
				</svg>
			</div>
			<div class="nb-title text-3xl mb-1">{node_count}</div>
			<div class="nb-text-muted">compute nodes</div>
		</div>
		
		<div class="nb-card-lg text-center">
			<div class="w-16 h-16 mx-auto mb-4 bg-[var(--nb-success)] rounded-full flex items-center justify-center">
				<svg class="w-8 h-8 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10"></path>
				</svg>
			</div>
			<div class="nb-title text-3xl mb-1">{container_count}</div>
			<div class="nb-text-muted">running containers</div>
		</div>
	</div>

	<div class="nb-card-lg">
		<div class="flex border-b-2 border-[var(--nb-border)] mb-6">
			<button 
				class="nb-button {activeTab === 'nodes' ? 'nb-button-primary' : ''} rounded-none border-b-0"
				on:click={() => activeTab = 'nodes'}
			>
				<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2m-2-4h.01M17 16h.01"></path>
				</svg>
				nodes
			</button>
			<button 
				class="nb-button {activeTab === 'users' ? 'nb-button-primary' : ''} rounded-none border-b-0"
				on:click={() => activeTab = 'users'}
			>
				<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4.354a4 4 0 110 5.292M15 21H3v-1a6 6 0 0112 0v1zm0 0h6v-1a6 6 0 00-9-5.197m13.5-9a2.5 2.5 0 11-5 0 2.5 2.5 0 015 0z"></path>
				</svg>
				users
			</button>
		</div>
		{#if activeTab === 'nodes'}
			<div class="flex items-center justify-between mb-6">
				<h2 class="nb-title text-2xl">node management</h2>
				<button class="nb-button nb-button-primary" on:click={() => showNodeModal = true}>
					<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6v6m0 0v6m0-6h6m-6 0H6"></path>
					</svg>
					add node
				</button>
			</div>
			
			{#if nodes.length}
				<div class="grid gap-4">
					{#each nodes as node}
						<div class="nb-card">
							<div class="flex items-center justify-between">
								<div class="flex items-center gap-4">
									<div class="w-12 h-12 bg-[var(--nb-secondary)] rounded-lg flex items-center justify-center">
										<svg class="w-6 h-6 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2m-2-4h.01M17 16h.01"></path>
										</svg>
									</div>
									<div>
										<h3 class="font-bold nb-mono">{node.name}</h3>
										<div class="text-sm nb-text-muted">
											<div>{node.hostname} {#if node.public_hostname}→ {node.public_hostname}{/if}</div>
											<div>{node.max_memory_mb}MB / {node.max_cpu_cores} cores / {node.max_storage_gb}GB</div>
										</div>
									</div>
								</div>
								
								<div class="flex items-center gap-3">
									<div class="text-right text-sm">
										<div class="nb-pill {node.is_online ? 'nb-pill-success' : 'nb-pill-danger'}">
											{node.is_online ? 'online' : 'offline'}
										</div>
										<div class="nb-text-muted mt-1">
											{node.last_seen ? new Date(node.last_seen).toLocaleString() : 'never seen'}
										</div>
									</div>
									
									<div class="flex gap-2">
										<button class="nb-button nb-button-sm nb-button-secondary" on:click={() => generateToken(node.id)}>
											<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
												<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 7a2 2 0 012 2m4 0a6 6 0 01-7.743 5.743L11 17H9v2H7v2H4a1 1 0 01-1-1v-2.586a1 1 0 01.293-.707l5.964-5.964A6 6 0 1721 9z"></path>
											</svg>
											new token
										</button>
										<button class="nb-button nb-button-sm nb-button-danger" on:click={() => deleteNode(node.id)}>
											<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
												<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"></path>
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
					<div class="w-20 h-20 mx-auto mb-4 bg-[var(--nb-muted)] rounded-full flex items-center justify-center">
						<svg class="w-10 h-10 nb-text-muted" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2m-2-4h.01M17 16h.01"></path>
						</svg>
					</div>
					<h3 class="nb-title text-xl mb-2">no nodes yet</h3>
					<p class="nb-text-muted mb-6">add compute nodes to start hosting containers</p>
				</div>
			{/if}
		{/if}

		{#if activeTab === 'users'}
			<div class="mb-6">
				<h2 class="nb-title text-2xl">user management</h2>
			</div>
			
			{#if users.length}
				<div class="grid gap-4">
					{#each users as user}
						<div class="nb-card">
							<div class="flex items-center justify-between">
								<div class="flex items-center gap-4">
									<div class="w-12 h-12 bg-[var(--nb-primary)] rounded-lg flex items-center justify-center">
										<svg class="w-6 h-6 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z"></path>
										</svg>
									</div>
									<div>
										<div class="flex items-center gap-2">
											<h3 class="font-bold">{user.display_name}</h3>
											{#if user.is_admin}
												<div class="nb-pill nb-pill-warning">admin</div>
											{/if}
										</div>
										<div class="text-sm nb-text-muted">
											<div>@{user.username} • {user.email}</div>
											<div>joined {new Date(user.created_at).toLocaleDateString()}</div>
										</div>
									</div>
								</div>
								
								<div class="flex items-center gap-3">
									<div class="text-right text-sm">
										{#if user.container_id}
											<div class="nb-pill nb-pill-success">has container</div>
											<div class="nb-text-muted nb-mono mt-1">{user.container_id}</div>
										{:else}
											<div class="nb-pill">no container</div>
										{/if}
									</div>
									
									<div class="flex gap-2">
										{#if user.container_id}
											<button class="nb-button nb-button-sm nb-button-warning" on:click={() => deleteUserContainer(user.id)}>
												<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
													<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10"></path>
												</svg>
												delete container
											</button>
										{/if}
										<button class="nb-button nb-button-sm nb-button-danger" on:click={() => deleteUser(user.id)}>
											<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
												<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"></path>
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
					<div class="w-20 h-20 mx-auto mb-4 bg-[var(--nb-muted)] rounded-full flex items-center justify-center">
						<svg class="w-10 h-10 nb-text-muted" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4.354a4 4 0 110 5.292M15 21H3v-1a6 6 0 0112 0v1zm0 0h6v-1a6 6 0 00-9-5.197m13.5-9a2.5 2.5 0 11-5 0 2.5 2.5 0 015 0z"></path>
						</svg>
					</div>
					<h3 class="nb-title text-xl mb-2">no users yet</h3>
					<p class="nb-text-muted">users will appear here when they sign up</p>
				</div>
			{/if}
		{/if}
	</div>
</main>

<Modal 
	show={showNodeModal} 
	title="add compute node" 
	size="lg"
	onClose={() => showNodeModal = false}
>
	<form on:submit|preventDefault={createNode} class="space-y-4">
		<div class="grid md:grid-cols-2 gap-4">
			<div>
				<label class="nb-label">node name</label>
				<input type="text" bind:value={newNode.name} required class="nb-input" placeholder="node-1">
			</div>
			<div>
				<label class="nb-label">internal hostname</label>
				<input type="text" bind:value={newNode.hostname} required class="nb-input" placeholder="192.168.1.100">
			</div>
		</div>
		
		<div>
			<label class="nb-label">public hostname (optional)</label>
			<input type="text" bind:value={newNode.public_hostname} class="nb-input" placeholder="node1.den.dev">
		</div>
		
		<div class="grid grid-cols-3 gap-4">
			<div>
				<label class="nb-label">memory (MB)</label>
				<input type="number" bind:value={newNode.max_memory_mb} class="nb-input">
			</div>
			<div>
				<label class="nb-label">cpu cores</label>
				<input type="number" bind:value={newNode.max_cpu_cores} class="nb-input">
			</div>
			<div>
				<label class="nb-label">storage (GB)</label>
				<input type="number" bind:value={newNode.max_storage_gb} class="nb-input">
			</div>
		</div>
	</form>
	
	<div slot="footer">
		<button class="nb-button nb-button-secondary" on:click={() => showNodeModal = false}>cancel</button>
		<button class="nb-button nb-button-primary" on:click={createNode}>create node</button>
	</div>
</Modal>

<Modal 
	show={showTokenModal} 
	title="node authentication token" 
	onClose={() => showTokenModal = false}
>
	<div class="text-center py-4">
		<div class="w-16 h-16 mx-auto mb-4 bg-[var(--nb-success)] rounded-full flex items-center justify-center">
			<svg class="w-8 h-8 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 7a2 2 0 012 2m4 0a6 6 0 01-7.743 5.743L11 17H9v2H7v2H4a1 1 0 01-1-1v-2.586a1 1 0 01.293-.707l5.964-5.964A6 6 0 1721 9z"></path>
			</svg>
		</div>
		<h3 class="nb-title text-xl mb-2">token generated!</h3>
		<p class="nb-text-muted mb-6">copy this token and use it to configure your node. it will not be shown again.</p>
		
		<div class="nb-card bg-[var(--nb-accent)] text-[var(--nb-success)] nb-mono text-sm p-4 break-all">
			{currentToken}
		</div>
	</div>
	
	<div slot="footer">
		<button class="nb-button nb-button-secondary" on:click={copyToken}>
			<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z"></path>
			</svg>
			copy token
		</button>
		<button class="nb-button nb-button-primary" on:click={() => showTokenModal = false}>done</button>
	</div>
</Modal>

<ToastContainer bind:this={toastContainer} />

