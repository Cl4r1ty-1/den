<script>
	import Header from '../lib/Header.svelte'
	export let user
	export let container
	export let subdomains = []

	let showSubdomainModal = false
	let newSubdomain = { subdomain: '', target_port: '' }

	async function createContainer() {
		const res = await fetch('/user/container/create', { method: 'POST', headers: { 'Content-Type': 'application/json' } })
		const data = await res.json()
		if (data.error) {
			alert('Error: ' + data.error)
			return
		}
		location.reload()
	}
	
	async function getNewPort() {
		const res = await fetch('/user/container/ports/new', { method: 'POST', headers: { 'Content-Type': 'application/json' } })
		const data = await res.json()
		if (data.error) {
			alert('Error: ' + data.error)
			return
		}
		location.reload()
	}

	async function createSubdomain() {
		const res = await fetch('/user/subdomains', {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify(newSubdomain)
		})
		const data = await res.json()
		if (data.error) {
			alert('Error: ' + data.error)
			return
		}
		showSubdomainModal = false
		newSubdomain = { subdomain: '', target_port: '' }
		location.reload()
	}

	async function deleteSubdomain(id) {
		if (!confirm('Delete this subdomain?')) return
		const res = await fetch(`/user/subdomains/${id}`, { method: 'DELETE' })
		const data = await res.json()
		if (data.error) {
			alert('Error: ' + data.error)
			return
		}
		location.reload()
	}
</script>

<Header {user} />
<div class="nb-container p-6">
	<h1 class="nb-title text-3xl mb-4">welcome, {user?.DisplayName}</h1>
	<div class="grid md:grid-cols-2 gap-4">
		<div class="nb-card">
			<h2 class="font-semibold mb-2">status</h2>
			{#if container}
				<div class="nb-pill">running</div>
				<div class="mt-3 text-sm text-slate-700 space-y-1">
					<div><span class="font-semibold">container id:</span> {container.ID}</div>
					<div><span class="font-semibold">ip:</span> {container.IPAddress}</div>
					<div><span class="font-semibold">memory:</span> {container.MemoryMB}MB</div>
					<div><span class="font-semibold">cpu:</span> {container.CPUCores}</div>
					<div><span class="font-semibold">storage:</span> {container.StorageGB}GB</div>
				</div>
				<div class="mt-3 flex gap-2">
					<button class="nb-button" on:click={getNewPort}>get new port</button>
					<a class="nb-button" href="/user/ssh-setup">configure ssh</a>
				</div>
			{:else}
				<div class="nb-pill">no environment</div>
				<p class="text-sm mt-2">create an environment to get started.</p>
				<div class="mt-3">
					<button class="nb-button" on:click={createContainer}>create container</button>
				</div>
			{/if}
		</div>
		<div class="nb-card">
			<div class="flex justify-between items-center mb-2">
				<h2 class="font-semibold">subdomains</h2>
				<button class="nb-button text-sm" on:click={() => showSubdomainModal = true}>add subdomain</button>
			</div>
			{#if subdomains.length}
				<ul class="space-y-2">
					{#each subdomains as s}
						<li class="flex justify-between items-center bg-[var(--nb-muted)] rounded-md p-2 border-2 border-[var(--nb-accent)]">
							<div class="font-mono text-sm">{s.Subdomain} → {s.TargetPort}</div>
							<button class="nb-button text-xs bg-red-500" on:click={() => deleteSubdomain(s.ID)}>delete</button>
						</li>
					{/each}
				</ul>
			{:else}
				<div class="text-sm text-slate-600">no subdomains yet.</div>
			{/if}
		</div>
	</div>
</div>

<!-- Add Subdomain Modal -->
{#if showSubdomainModal}
	<div class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
		<div class="nb-card max-w-md w-full mx-4">
			<h3 class="text-lg font-semibold mb-4">add subdomain</h3>
			<form on:submit|preventDefault={createSubdomain}>
				<div class="space-y-3">
					<div>
						<label class="block text-sm font-medium mb-1">subdomain</label>
						<input type="text" bind:value={newSubdomain.subdomain} required placeholder="myapp" class="w-full p-2 border-2 border-[var(--nb-accent)] rounded-md">
						<div class="text-xs text-slate-600 mt-1">will be available at {newSubdomain.subdomain}.den.dev</div>
					</div>
					<div>
						<label class="block text-sm font-medium mb-1">target port</label>
						<input type="number" bind:value={newSubdomain.target_port} required placeholder="3000" class="w-full p-2 border-2 border-[var(--nb-accent)] rounded-md">
					</div>
				</div>
				<div class="flex gap-2 mt-4">
					<button type="submit" class="nb-button">create</button>
					<button type="button" class="nb-button bg-gray-500" on:click={() => showSubdomainModal = false}>cancel</button>
				</div>
			</form>
		</div>
	</div>
{/if}

