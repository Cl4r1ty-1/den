<script>
	import Header from '../lib/Header.svelte'
	import Modal from '../lib/Modal.svelte'
	import ToastContainer from '../lib/ToastContainer.svelte'
	
	export let user
	export let container
	export let subdomains = []

	let showSubdomainModal = false
	let newSubdomain = { subdomain: '', target_port: '', subdomain_type: 'project' }
	let toastContainer

	async function getNewPort() {
		const res = await fetch('/user/container/ports/new', { method: 'POST', headers: { 'Content-Type': 'application/json' } })
		const data = await res.json()
		if (data.error) {
			toastContainer.addToast(data.error, 'danger')
			return
		}
		toastContainer.addToast(`Allocated port: ${data.port}`, 'success')
		setTimeout(() => location.reload(), 1000)
	}

	async function createSubdomain() {
		const res = await fetch('/user/subdomains', {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify(newSubdomain)
		})
		const data = await res.json()
		if (data.error) {
			toastContainer.addToast(data.error, 'danger')
			return
		}
		toastContainer.addToast('Subdomain created successfully!', 'success')
		showSubdomainModal = false
		newSubdomain = { subdomain: '', target_port: '', subdomain_type: 'project' }
		setTimeout(() => location.reload(), 1000)
	}

	async function deleteSubdomain(id) {
		const res = await fetch(`/user/subdomains/${id}`, { method: 'DELETE' })
		const data = await res.json()
		if (data.error) {
			toastContainer.addToast(data.error, 'danger')
			return
		}
		toastContainer.addToast('Subdomain deleted', 'success')
		setTimeout(() => location.reload(), 1000)
	}
</script>

<Header {user} currentPage="subdomains" />

<main class="nb-container py-8">
	<div class="mb-8">
		<h1 class="nb-title text-4xl mb-2">
			<span class="text-[var(--nb-primary)]">subdomain</span> management
		</h1>
		<p class="nb-subtitle text-xl">expose your applications to the internet</p>
	</div>

	{#if container}
		<div class="grid md:grid-cols-3 gap-6 mb-8">
			<div class="nb-card text-center">
				<div class="w-12 h-12 mx-auto mb-3 bg-[var(--nb-info)] rounded-lg flex items-center justify-center">
					<svg class="w-6 h-6 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 12a9 9 0 01-9 9m9-9a9 9 0 00-9-9m9 9H3m9 9a9 9 0 01-9-9m9 9c1.657 0 3-4.03 3-9s-1.343-9-3-9m0 18c-1.657 0-3-4.03-3-9s1.343-9 3-9m-9 9a9 9 0 019-9"></path>
					</svg>
				</div>
				<div class="nb-title text-2xl">{subdomains.length}</div>
				<div class="nb-text-muted text-sm">active subdomains</div>
			</div>
			
			<div class="nb-card text-center">
				<div class="w-12 h-12 mx-auto mb-3 bg-[var(--nb-secondary)] rounded-lg flex items-center justify-center">
					<svg class="w-6 h-6 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 9l3 3-3 3m5 0h3M5 20h14a2 2 0 002-2V6a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z"></path>
					</svg>
				</div>
				<div class="nb-title text-2xl">{container?.AllocatedPorts?.length || 0}</div>
				<div class="nb-text-muted text-sm">available ports</div>
			</div>
			
			<div class="nb-card text-center">
				<div class="w-12 h-12 mx-auto mb-3 bg-[var(--nb-success)] rounded-lg flex items-center justify-center">
					<svg class="w-6 h-6 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z"></path>
					</svg>
				</div>
				<div class="nb-title text-2xl">SSL</div>
				<div class="nb-text-muted text-sm">auto certificates</div>
			</div>
		</div>
	{/if}

	<div class="nb-card-lg">
		<div class="flex items-center justify-between mb-6">
			<h2 class="nb-title text-2xl">your subdomains</h2>
			{#if container}
				<button class="nb-button nb-button-primary" on:click={() => showSubdomainModal = true}>
					<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6v6m0 0v6m0-6h6m-6 0H6"></path>
					</svg>
					create subdomain
				</button>
			{/if}
		</div>
		
		{#if subdomains.length}
			<div class="grid gap-4">
				{#each subdomains as subdomain}
					<div class="nb-card">
						<div class="flex items-center justify-between">
							<div class="flex items-center gap-4">
								<div class="w-12 h-12 bg-[var(--nb-primary)] rounded-lg flex items-center justify-center">
									<svg class="w-6 h-6 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 12a9 9 0 01-9 9m9-9a9 9 0 00-9-9m9 9H3m9 9a9 9 0 01-9-9m9 9c1.657 0 3-4.03 3-9s-1.343-9-3-9m0 18c-1.657 0-3-4.03-3-9s1.343-9 3-9m-9 9a9 9 0 019-9"></path>
									</svg>
								</div>
								<div>
									<div class="flex items-center gap-2 mb-1">
										<div class="nb-mono font-bold text-lg">
											{#if subdomain.SubdomainType === 'username'}
												{subdomain.Subdomain}.hack.kim
											{:else}
												{subdomain.Subdomain}.{user.Username}.hack.kim
											{/if}
										</div>
										<div class="nb-pill {subdomain.IsActive ? 'nb-pill-success' : 'nb-pill-danger'}">
											{subdomain.IsActive ? 'active' : 'inactive'}
										</div>
									</div>
									<div class="text-sm nb-text-muted">
										<div>→ port {subdomain.TargetPort}</div>
										<div>created {new Date(subdomain.CreatedAt).toLocaleDateString()}</div>
									</div>
								</div>
							</div>
							
							<div class="flex items-center gap-3">
								<a 
									href="https://{subdomain.SubdomainType === 'username' ? subdomain.Subdomain + '.hack.kim' : subdomain.Subdomain + '.' + user.Username + '.hack.kim'}" 
									target="_blank"
									class="nb-button nb-button-sm nb-button-secondary"
								>
									<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14"></path>
									</svg>
									visit
								</a>
								<button class="nb-button nb-button-sm nb-button-danger" on:click={() => deleteSubdomain(subdomain.ID)}>
									<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"></path>
									</svg>
									delete
								</button>
							</div>
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
		<div class="nb-card-lg mt-8">
			<div class="flex items-center justify-between mb-6">
				<h2 class="nb-title text-xl">allocated ports</h2>
				<button class="nb-button nb-button-secondary" on:click={getNewPort}>
					<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6v6m0 0v6m0-6h6m-6 0H6"></path>
					</svg>
					get new port
				</button>
			</div>
			
			{#if container.AllocatedPorts && container.AllocatedPorts.length}
				<div class="grid grid-cols-2 md:grid-cols-4 lg:grid-cols-6 gap-4">
					{#each container.AllocatedPorts as port}
						<div class="nb-card text-center">
							<div class="nb-title text-lg">{port}</div>
							<div class="text-xs nb-text-muted">available</div>
						</div>
					{/each}
				</div>
			{:else}
				<div class="text-center py-8">
					<div class="w-16 h-16 mx-auto mb-4 bg-[var(--nb-muted)] rounded-full flex items-center justify-center">
						<svg class="w-8 h-8 nb-text-muted" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 9l3 3-3 3m5 0h3M5 20h14a2 2 0 002-2V6a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z"></path>
						</svg>
					</div>
					<h3 class="nb-title text-lg mb-2">no ports allocated</h3>
					<p class="nb-text-muted">request a port to start hosting applications</p>
				</div>
			{/if}
		</div>
	{/if}

	<div class="nb-card bg-[var(--nb-info)] text-white mt-8">
		<div class="flex items-start gap-3">
			<svg class="w-5 h-5 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path>
			</svg>
			<div>
				<h4 class="font-bold mb-2">how subdomains work</h4>
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

<Modal 
	show={showSubdomainModal} 
	title="create subdomain" 
	onClose={() => showSubdomainModal = false}
>
	<form on:submit|preventDefault={createSubdomain} class="space-y-4">
		<div>
			<label class="nb-label">subdomain type</label>
			<div class="space-y-2">
				<label class="flex items-center gap-2 cursor-pointer">
					<input type="radio" bind:group={newSubdomain.subdomain_type} value="username" class="w-4 h-4">
					<span>username subdomain ({user.Username}.hack.kim)</span>
				</label>
				<label class="flex items-center gap-2 cursor-pointer">
					<input type="radio" bind:group={newSubdomain.subdomain_type} value="project" class="w-4 h-4">
					<span>project subdomain (myapp.{user.Username}.hack.kim)</span>
				</label>
			</div>
		</div>
		
		<div>
			<label class="nb-label">subdomain name</label>
			<input 
				type="text" 
				bind:value={newSubdomain.subdomain} 
				required 
				placeholder={newSubdomain.subdomain_type === 'username' ? user.Username : 'myapp'}
				class="nb-input"
			>
			<div class="text-xs nb-text-muted mt-1">
				preview: {newSubdomain.subdomain || (newSubdomain.subdomain_type === 'username' ? user.Username : 'myapp')}{newSubdomain.subdomain_type === 'username' ? '.hack.kim' : '.' + user.Username + '.hack.kim'}
			</div>
		</div>
		
		<div>
			<label class="nb-label">target port</label>
			<select bind:value={newSubdomain.target_port} required class="nb-input nb-select">
				<option value="">select a port</option>
				{#if container?.AllocatedPorts}
					{#each container.AllocatedPorts as port}
						<option value={port}>{port}</option>
					{/each}
				{/if}
			</select>
		</div>
	</form>
	
	<div slot="footer">
		<button class="nb-button nb-button-secondary" on:click={() => showSubdomainModal = false}>cancel</button>
		<button class="nb-button nb-button-primary" on:click={createSubdomain}>create subdomain</button>
	</div>
</Modal>

<ToastContainer bind:this={toastContainer} />
