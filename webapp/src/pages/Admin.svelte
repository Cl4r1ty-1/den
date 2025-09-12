<script>
	import Header from '../lib/Header.svelte'
	export let user_count = 0
	export let node_count = 0
	export let container_count = 0

	let nodes = []
	let users = []
	let showNodeModal = false
	let showUserModal = false
	let newNode = { name: '', hostname: '', public_hostname: '', max_memory_mb: 4096, max_cpu_cores: 4, max_storage_gb: 15 }

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
			alert('Error: ' + data.error)
			return
		}
		showNodeModal = false
		newNode = { name: '', hostname: '', public_hostname: '', max_memory_mb: 4096, max_cpu_cores: 4, max_storage_gb: 15 }
		loadNodes()
		alert('Node created! Token: ' + data.token)
	}

	async function generateToken(nodeId) {
		if (!confirm('Generate new token? Old token will be invalidated.')) return
		const res = await fetch(`/admin/nodes/${nodeId}/token`, { method: 'GET' })
		const data = await res.json()
		if (data.error) {
			alert('Error: ' + data.error)
			return
		}
		alert('New token: ' + data.token)
	}

	async function deleteNode(nodeId) {
		if (!confirm('Delete this node? All containers on this node will be lost!')) return
		const res = await fetch(`/admin/nodes/${nodeId}`, { method: 'DELETE' })
		const data = await res.json()
		if (data.error) {
			alert('Error: ' + data.error)
			return
		}
		loadNodes()
	}

	async function deleteUser(userId) {
		if (!confirm('Delete this user? Their container and data will be lost.')) return
		const res = await fetch(`/admin/users/${userId}`, { method: 'DELETE' })
		const data = await res.json()
		if (data.error) {
			alert('Error: ' + data.error)
			return
		}
		loadUsers()
	}

	async function deleteUserContainer(userId) {
		if (!confirm('Delete this user\'s container? Their data will be lost.')) return
		const res = await fetch(`/admin/users/${userId}/container`, { method: 'DELETE' })
		const data = await res.json()
		if (data.error) {
			alert('Error: ' + data.error)
			return
		}
		loadUsers()
	}

	loadNodes()
	loadUsers()
</script>

<Header user={{ IsAdmin: true }} />
<div class="nb-container p-6">
	<h1 class="nb-title text-3xl mb-4">admin</h1>
	
	<!-- Stats -->
	<div class="grid md:grid-cols-3 gap-4 mb-8">
		<div class="nb-card text-center">
			<div class="text-4xl font-extrabold">{user_count}</div>
			<div class="text-slate-600">users</div>
		</div>
		<div class="nb-card text-center">
			<div class="text-4xl font-extrabold">{node_count}</div>
			<div class="text-slate-600">nodes</div>
		</div>
		<div class="nb-card text-center">
			<div class="text-4xl font-extrabold">{container_count}</div>
			<div class="text-slate-600">containers</div>
		</div>
	</div>

	<!-- Nodes Table -->
	<div class="nb-card mb-6">
		<div class="flex justify-between items-center mb-4">
			<h2 class="text-xl font-semibold">nodes</h2>
			<button class="nb-button" on:click={() => showNodeModal = true}>add node</button>
		</div>
		<div class="overflow-x-auto">
			<table class="w-full">
				<thead>
					<tr class="border-b-2 border-[var(--nb-accent)]">
						<th class="text-left p-2">name</th>
						<th class="text-left p-2">hostname</th>
						<th class="text-left p-2">public</th>
						<th class="text-left p-2">resources</th>
						<th class="text-left p-2">status</th>
						<th class="text-left p-2">last seen</th>
						<th class="text-left p-2">actions</th>
					</tr>
				</thead>
				<tbody>
					{#each nodes as node}
						<tr class="border-b border-[var(--nb-muted)]">
							<td class="p-2 font-mono">{node.name}</td>
							<td class="p-2 font-mono">{node.hostname}</td>
							<td class="p-2 font-mono">{node.public_hostname || 'none'}</td>
							<td class="p-2 text-sm">{node.max_memory_mb}mb / {node.max_cpu_cores} cores / {node.max_storage_gb}gb</td>
							<td class="p-2">
								<span class="nb-pill {node.is_online ? 'bg-green-100' : 'bg-red-100'}">
									{node.is_online ? 'online' : 'offline'}
								</span>
							</td>
							<td class="p-2 text-sm">{node.last_seen ? new Date(node.last_seen).toLocaleString() : 'never'}</td>
							<td class="p-2">
								<div class="flex gap-2">
									<button class="nb-button text-xs" on:click={() => generateToken(node.id)}>new token</button>
									<button class="nb-button text-xs bg-red-500" on:click={() => deleteNode(node.id)}>delete</button>
								</div>
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	</div>

	<!-- Users Table -->
	<div class="nb-card">
		<div class="flex justify-between items-center mb-4">
			<h2 class="text-xl font-semibold">users</h2>
		</div>
		<div class="overflow-x-auto">
			<table class="w-full">
				<thead>
					<tr class="border-b-2 border-[var(--nb-accent)]">
						<th class="text-left p-2">username</th>
						<th class="text-left p-2">email</th>
						<th class="text-left p-2">display name</th>
						<th class="text-left p-2">admin</th>
						<th class="text-left p-2">container</th>
						<th class="text-left p-2">created</th>
						<th class="text-left p-2">actions</th>
					</tr>
				</thead>
				<tbody>
					{#each users as user}
						<tr class="border-b border-[var(--nb-muted)]">
							<td class="p-2 font-mono">{user.username}</td>
							<td class="p-2">{user.email}</td>
							<td class="p-2">{user.display_name}</td>
							<td class="p-2">
								{#if user.is_admin}
									<span class="nb-pill bg-blue-100">admin</span>
								{/if}
							</td>
							<td class="p-2 font-mono">{user.container_id || 'none'}</td>
							<td class="p-2 text-sm">{new Date(user.created_at).toLocaleDateString()}</td>
							<td class="p-2">
								<div class="flex gap-2">
									{#if user.container_id}
										<button class="nb-button text-xs bg-orange-500" on:click={() => deleteUserContainer(user.id)}>delete container</button>
									{/if}
									<button class="nb-button text-xs bg-red-500" on:click={() => deleteUser(user.id)}>delete user</button>
								</div>
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	</div>
</div>

<!-- Add Node Modal -->
{#if showNodeModal}
	<div class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
		<div class="nb-card max-w-md w-full mx-4">
			<h3 class="text-lg font-semibold mb-4">add node</h3>
			<form on:submit|preventDefault={createNode}>
				<div class="space-y-3">
					<div>
						<label class="block text-sm font-medium mb-1">name</label>
						<input type="text" bind:value={newNode.name} required class="w-full p-2 border-2 border-[var(--nb-accent)] rounded-md">
					</div>
					<div>
						<label class="block text-sm font-medium mb-1">hostname</label>
						<input type="text" bind:value={newNode.hostname} required class="w-full p-2 border-2 border-[var(--nb-accent)] rounded-md">
					</div>
					<div>
						<label class="block text-sm font-medium mb-1">public hostname (optional)</label>
						<input type="text" bind:value={newNode.public_hostname} class="w-full p-2 border-2 border-[var(--nb-accent)] rounded-md">
					</div>
					<div class="grid grid-cols-3 gap-2">
						<div>
							<label class="block text-sm font-medium mb-1">memory (mb)</label>
							<input type="number" bind:value={newNode.max_memory_mb} class="w-full p-2 border-2 border-[var(--nb-accent)] rounded-md">
						</div>
						<div>
							<label class="block text-sm font-medium mb-1">cpu cores</label>
							<input type="number" bind:value={newNode.max_cpu_cores} class="w-full p-2 border-2 border-[var(--nb-accent)] rounded-md">
						</div>
						<div>
							<label class="block text-sm font-medium mb-1">storage (gb)</label>
							<input type="number" bind:value={newNode.max_storage_gb} class="w-full p-2 border-2 border-[var(--nb-accent)] rounded-md">
						</div>
					</div>
				</div>
				<div class="flex gap-2 mt-4">
					<button type="submit" class="nb-button">create</button>
					<button type="button" class="nb-button bg-gray-500" on:click={() => showNodeModal = false}>cancel</button>
				</div>
			</form>
		</div>
	</div>
{/if}

