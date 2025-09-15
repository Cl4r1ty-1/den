<script lang="ts">
	import Header from '../lib/Header.svelte'
	import Modal from '../lib/Modal.svelte'
	import ToastContainer from '../lib/ToastContainer.svelte'
	
	type Container = { allocated_ports?: number[] }
	type Subdomain = { id: number; subdomain: string; target_port: number; subdomain_type: 'project'|'username'; is_active?: boolean; created_at?: string }
	export let user: { username: string }
	export let container: Container | null = null
	export let subdomains: Subdomain[] = []

	let showSubdomainModal = false
	let newSubdomain: { subdomain: string; target_port: string|number; subdomain_type: 'project'|'username' } = { subdomain: '', target_port: '', subdomain_type: 'project' }
	let toastContainer: any

	async function getNewPort() {
		const res = await fetch('/user/container/ports/new', { method: 'POST', headers: { 'Content-Type': 'application/json' } })
		const data = await res.json()
		if (data.error) { toastContainer.addToast(data.error, 'danger'); return }
		toastContainer.addToast(`Allocated port: ${data.port}`, 'success')
		setTimeout(() => location.reload(), 1000)
	}

	async function createSubdomain() {
		if (newSubdomain.subdomain_type === 'username') { newSubdomain.subdomain = user.username }
		const res = await fetch('/user/subdomains', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(newSubdomain) })
		const data = await res.json()
		if (data.error) { toastContainer.addToast(data.error, 'danger'); return }
		toastContainer.addToast('Subdomain created successfully!', 'success')
		showSubdomainModal = false
		newSubdomain = { subdomain: '', target_port: '', subdomain_type: 'project' }
		setTimeout(() => location.reload(), 1000)
	}

	async function deleteSubdomain(id: number) {
		const res = await fetch(`/user/subdomains/${id}`, { method: 'DELETE' })
		const data = await res.json()
		if (data.error) { toastContainer.addToast(data.error, 'danger'); return }
		toastContainer.addToast('Subdomain deleted', 'success')
		setTimeout(() => location.reload(), 1000)
	}

	$: if (newSubdomain.subdomain_type === 'username') { newSubdomain.subdomain = user?.username || '' }
</script>

<div class="min-h-screen bg-background text-foreground">
	<Header {user} currentPage="subdomains" />

	<main class="max-w-6xl mx-auto p-6">
		<div class="mb-8">
			<h1 class="text-4xl font-heading mb-2">
				<span class="text-main">subdomain</span> management
			</h1>
			<p class="text-xl text-foreground/70">expose your applications to the internet</p>
		</div>

		{#if container}
			<div class="grid md:grid-cols-3 gap-6 mb-8">
				<div class="bg-secondary-background border-2 border-border p-6 text-center shadow-shadow">
					<div class="w-12 h-12 mx-auto mb-3 bg-chart-2 border-2 border-border flex items-center justify-center">
						<svg class="w-6 h-6 text-main-foreground" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 12a9 9 0 01-9 9m9-9a9 9 0 00-9-9m9 9H3m9 9a9 9 0 01-9-9m9 9c1.657 0 3-4.03 3-9s-1.343-9-3-9m0 18c-1.657 0-3-4.03-3-9s1.343-9 3-9m-9 9a9 9 0 019-9"></path>
						</svg>
					</div>
					<div class="text-2xl font-heading">{subdomains.length}</div>
					<div class="text-foreground/70 text-sm">active subdomains</div>
				</div>
				
				<div class="bg-secondary-background border-2 border-border p-6 text-center shadow-shadow">
					<div class="w-12 h-12 mx-auto mb-3 bg-chart-3 border-2 border-border flex items-center justify-center">
						<svg class="w-6 h-6 text-main-foreground" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 9l3 3-3 3m5 0h3M5 20h14a2 2 0 002-2V6a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z"></path>
						</svg>
					</div>
					<div class="text-2xl font-heading">{container?.allocated_ports?.length || 0}</div>
					<div class="text-foreground/70 text-sm">available ports</div>
				</div>
				
				<div class="bg-secondary-background border-2 border-border p-6 text-center shadow-shadow">
					<div class="w-12 h-12 mx-auto mb-3 bg-chart-4 border-2 border-border flex items-center justify-center">
						<svg class="w-6 h-6 text-main-foreground" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z"></path>
						</svg>
					</div>
					<div class="text-2xl font-heading">SSL</div>
					<div class="text-foreground/70 text-sm">auto certificates</div>
				</div>
			</div>
		{/if}

		<div class="bg-secondary-background border-2 border-border p-6 shadow-shadow">
			<div class="flex items-center justify-between mb-6">
				<h2 class="text-2xl font-heading">your subdomains</h2>
				{#if container}
					<button class="bg-main text-main-foreground border-2 border-border px-4 py-2 font-heading hover:translate-x-1 hover:translate-y-1 transition-transform shadow-shadow" on:click={() => showSubdomainModal = true}>
						<svg class="w-4 h-4 inline mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6v6m0 0v6m0-6h6m-6 0H6"></path>
						</svg>
						create subdomain
					</button>
				{/if}
			</div>
		
			{#if subdomains.length}
				<div class="grid gap-4">
					{#each subdomains as subdomain}
						<div class="bg-background border-2 border-border p-4 flex items-center justify-between shadow-shadow">
							<div class="flex items-center gap-4">
								<div class="w-12 h-12 bg-chart-2 border-2 border-border flex items-center justify-center">
									<svg class="w-6 h-6 text-main-foreground" fill="none" stroke="currentColor" viewBox="0 0 24 24">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 12a9 9 0 01-9 9m9-9a9 9 0 00-9-9m9 9H3m9 9a9 9 0 01-9-9m9 9c1.657 0 3-4.03 3-9s-1.343-9-3-9m0 18c-1.657 0-3-4.03-3-9s1.343-9 3-9m-9 9a9 9 0 919-9"></path>
									</svg>
								</div>
								<div>
									<div class="flex items-center gap-2 mb-1">
										<div class="font-mono font-bold text-lg">
											{#if subdomain.subdomain_type === 'username'}
												{subdomain.subdomain}.hack.kim
											{:else}
												{subdomain.subdomain}.{user.username}.hack.kim
											{/if}
										</div>
										<div class="px-2 py-1 text-xs border-2 border-border {subdomain.is_active ? 'bg-chart-4 text-main-foreground' : 'bg-chart-1 text-main-foreground'}">
											{subdomain.is_active ? 'active' : 'inactive'}
										</div>
									</div>
									<div class="text-sm text-foreground/70">
										<div>→ port {subdomain.target_port}</div>
										<div>created {new Date(subdomain.created_at).toLocaleDateString()}</div>
									</div>
								</div>
							</div>
							
							<div class="flex items-center gap-3">
								<a 
									href="https://{subdomain.subdomain_type === 'username' ? subdomain.subdomain + '.hack.kim' : subdomain.subdomain + '.' + user.username + '.hack.kim'}" 
									target="_blank"
									class="bg-chart-4 text-main-foreground border-2 border-border px-3 py-1 text-sm font-heading hover:translate-x-1 hover:translate-y-1 transition-transform shadow-shadow"
								>
									<svg class="w-4 h-4 inline mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14"></path>
									</svg>
									visit
								</a>
								<button class="bg-chart-1 text-main-foreground border-2 border-border px-3 py-1 text-sm font-heading hover:translate-x-1 hover:translate-y-1 transition-transform shadow-shadow" on:click={() => deleteSubdomain(subdomain.id)}>
									<svg class="w-4 h-4 inline mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"></path>
									</svg>
									delete
								</button>
							</div>
						</div>
					{/each}
				</div>
		{:else}
			<div class="text-center py-12">
				<div class="w-20 h-20 mx-auto mb-4 bg-[var(--nb-muted)] rounded-full flex items-center justify-center">
					<svg class="w-10 h-10 nb-text-muted" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 12a9 9 0 01-9 9m9-9a9 9 0 00-9-9m9 9H3m9 9a9 9 0 01-9-9m9 9c1.657 0 3-4.03 3-9s-1.343-9-3-9m0 18c-1.657 0-3-4.03-3-9s1.343-9 3-9m-9 9a9 9 0 019-9"></path>
					</svg>
				</div>
				<h3 class="nb-title text-xl mb-2">no subdomains yet</h3>
				<p class="nb-text-muted mb-6">create subdomains to expose your applications to the internet</p>
				{#if !container}
					<p class="text-sm nb-text-muted">create an environment first to manage subdomains</p>
				{/if}
			</div>
		{/if}
	</div>

		{#if container}
			<div class="bg-secondary-background border-2 border-border p-6 shadow-shadow mt-8">
				<div class="flex items-center justify-between mb-6">
					<h2 class="text-xl font-heading">allocated ports</h2>
					<button class="bg-chart-3 text-main-foreground border-2 border-border px-4 py-2 font-heading hover:translate-x-1 hover:translate-y-1 transition-transform shadow-shadow" on:click={getNewPort}>
						<svg class="w-4 h-4 inline mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6v6m0 0v6m0-6h6m-6 0H6"></path>
						</svg>
						get new port
					</button>
				</div>
			
				{#if container?.allocated_ports && container.allocated_ports.length}
					<div class="grid grid-cols-2 md:grid-cols-4 lg:grid-cols-6 gap-4">
						{#each container.allocated_ports as port}
							<div class="bg-background border-2 border-border p-4 text-center shadow-shadow">
								<div class="text-lg font-heading">{port}</div>
								<div class="text-xs text-foreground/70">available</div>
							</div>
						{/each}
					</div>
				{:else}
					<div class="text-center py-8">
						<div class="w-16 h-16 mx-auto mb-4 bg-foreground/10 border-2 border-border flex items-center justify-center">
							<svg class="w-8 h-8 text-foreground/50" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 9l3 3-3 3m5 0h3M5 20h14a2 2 0 002-2V6a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z"></path>
							</svg>
						</div>
						<h3 class="text-lg font-heading mb-2">no ports allocated</h3>
						<p class="text-foreground/70">request a port to start hosting applications</p>
					</div>
				{/if}
			</div>
		{/if}

		<div class="bg-chart-2 text-main-foreground border-2 border-border p-6 shadow-shadow mt-8">
			<div class="flex items-start gap-3">
				<svg class="w-5 h-5 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path>
				</svg>
				<div>
					<h4 class="font-heading font-bold mb-2">how subdomains work</h4>
					<ul class="text-sm opacity-90 space-y-1">
						<li>• <strong>username subdomains:</strong> yourname.hack.kim (only one allowed)</li>
						<li>• <strong>project subdomains:</strong> myapp.yourname.hack.kim (unlimited)</li>
						<li>• all subdomains get automatic SSL certificates</li>
						<li>• point to any port from your allocated range</li>
					</ul>
				</div>
			</div>
		</div>
	</main>
</div>

<Modal 
	show={showSubdomainModal} 
	title="create subdomain" 
	onClose={() => showSubdomainModal = false}
>
	<form on:submit|preventDefault={createSubdomain} class="space-y-4">
		<div>
			<label class="block text-sm font-heading mb-2" for="sub_type">subdomain type</label>
			<div id="sub_type" class="space-y-2">
				<label class="flex items-center gap-2 cursor-pointer">
					<input type="radio" bind:group={newSubdomain.subdomain_type} value="username" class="w-4 h-4">
					<span>username subdomain ({user.username}.hack.kim)</span>
				</label>
				<label class="flex items-center gap-2 cursor-pointer">
					<input type="radio" bind:group={newSubdomain.subdomain_type} value="project" class="w-4 h-4">
					<span>project subdomain (myapp.{user.username}.hack.kim)</span>
				</label>
			</div>
		</div>
		
		{#if newSubdomain.subdomain_type === 'username'}
			<div>
				<label class="block text-sm font-heading mb-2" for="dom_preview">domain</label>
				<div id="dom_preview" class="w-full bg-background border-2 border-border p-3">
					your project will be on <span class="font-mono font-bold">{user.username}.hack.kim</span>
				</div>
			</div>
		{:else}
			<div>
				<label class="block text-sm font-heading mb-2" for="sub_name">subdomain name</label>
				<input id="sub_name" type="text" bind:value={newSubdomain.subdomain} required class="w-full bg-background border-2 border-border p-3 font-mono" placeholder="my-app">
				<div class="text-xs text-foreground/70 mt-1">
					preview: {newSubdomain.subdomain || 'myapp'}.{user.username}.hack.kim
				</div>
			</div>
		{/if}
		
		<div>
			<label class="block text-sm font-heading mb-2" for="sub_port">target port</label>
			<select id="sub_port" bind:value={newSubdomain.target_port} required class="w-full bg-background border-2 border-border p-3">
				<option value="">select a port</option>
				{#if container?.allocated_ports}
					{#each container.allocated_ports as port}
						<option value={port}>{port}</option>
					{/each}
				{/if}
			</select>
		</div>
	</form>
	
	<div slot="footer" class="flex gap-3">
		<button class="bg-foreground/10 border-2 border-border px-4 py-2 font-heading hover:translate-x-1 hover:translate-y-1 transition-transform" on:click={() => showSubdomainModal = false}>cancel</button>
		<button class="bg-main text-main-foreground border-2 border-border px-4 py-2 font-heading hover:translate-x-1 hover:translate-y-1 transition-transform shadow-shadow" on:click={createSubdomain}>create subdomain</button>
	</div>
</Modal>

<ToastContainer bind:this={toastContainer} />
