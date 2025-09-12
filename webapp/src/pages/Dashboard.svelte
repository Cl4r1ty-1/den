<script>
	import Header from '../lib/Header.svelte'
	import Modal from '../lib/Modal.svelte'
	import ToastContainer from '../lib/ToastContainer.svelte'
	
	export let user
	export let container = null
	export let subdomains = []

	let showSubdomainModal = false
	let showContainerModal = false
	let newSubdomain = { subdomain: '', target_port: '', subdomain_type: 'project' }
	let toastContainer

	async function createContainer() {
		const res = await fetch('/user/container/create', { method: 'POST', headers: { 'Content-Type': 'application/json' } })
		const data = await res.json()
		if (data.error) {
			toastContainer.addToast(data.error, 'danger')
			return
		}
		toastContainer.addToast('Container creation started! Refresh in a few minutes.', 'success')
		showContainerModal = false
		setTimeout(() => location.reload(), 2000)
	}
	
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
	
	function updateSubdomainPreview() {
	}
</script>

<Header {user} currentPage="dashboard" />

<main class="nb-container py-8">
	<div class="mb-8">
		<h1 class="nb-title text-4xl mb-2">
			welcome back, <span class="text-[var(--nb-primary)]">{user?.DisplayName}</span>! 👋
		</h1>
		<p class="nb-subtitle text-xl">manage your cozy *nix environment</p>
	</div>

	<div class="grid lg:grid-cols-3 gap-6 mb-8">
		<div class="lg:col-span-2">
			<div class="nb-card-lg">
				<div class="flex items-center justify-between mb-4">
					<h2 class="nb-title text-2xl">environment status</h2>
					{#if container}
						<div class="nb-pill nb-pill-success">
							<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"></path>
							</svg>
							running
						</div>
					{:else}
						<div class="nb-pill nb-pill-danger">
							<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2m7-2a9 9 0 11-18 0 9 9 0 0118 0z"></path>
							</svg>
							no environment
						</div>
					{/if}
				</div>
				
				{#if container}
					<div class="grid md:grid-cols-2 gap-4 mb-6">
						<div class="nb-card">
							<h3 class="font-bold mb-2 text-[var(--nb-text-muted)]">container details</h3>
							<div class="space-y-2 text-sm">
								<div class="flex justify-between">
									<span class="nb-text-muted">ID:</span>
									<code class="nb-mono font-bold">{container.ID}</code>
								</div>
								<div class="flex justify-between">
									<span class="nb-text-muted">IP:</span>
									<code class="nb-mono font-bold">{container.IPAddress}</code>
								</div>
							</div>
						</div>
						
						<div class="nb-card">
							<h3 class="font-bold mb-2 text-[var(--nb-text-muted)]">resources</h3>
							<div class="space-y-2 text-sm">
								<div class="flex justify-between">
									<span class="nb-text-muted">Memory:</span>
									<span class="font-bold">{container.MemoryMB}MB</span>
								</div>
								<div class="flex justify-between">
									<span class="nb-text-muted">CPU:</span>
									<span class="font-bold">{container.CPUCores} cores</span>
								</div>
								<div class="flex justify-between">
									<span class="nb-text-muted">Storage:</span>
									<span class="font-bold">{container.StorageGB}GB</span>
								</div>
							</div>
						</div>
					</div>
					
					<div class="nb-card bg-[var(--nb-surface-alt)] mb-4">
						<h3 class="font-bold mb-2">ssh access</h3>
						<div class="nb-card bg-[var(--nb-accent)] text-[var(--nb-success)] nb-mono text-sm p-3">
							ssh {user.Username}@hack.kim
						</div>
					</div>
					
					<div class="mb-4">
						<div class="flex items-center justify-between mb-3">
							<h3 class="font-bold">allocated ports</h3>
							<button class="nb-button nb-button-sm nb-button-secondary" on:click={getNewPort}>
								<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6v6m0 0v6m0-6h6m-6 0H6"></path>
								</svg>
								get new port
							</button>
						</div>
						{#if container?.AllocatedPorts?.length}
							<div class="flex flex-wrap gap-2">
								{#each container.AllocatedPorts as port}
									<div class="nb-pill">{port}</div>
								{/each}
							</div>
						{:else}
							<p class="nb-text-muted text-sm">no ports allocated yet</p>
						{/if}
					</div>
					
					<div class="flex gap-3">
						<a href="/user/ssh-setup" class="nb-button nb-button-primary">
							<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 7a2 2 0 012 2m4 0a6 6 0 01-7.743 5.743L11 17H9v2H7v2H4a1 1 0 01-1-1v-2.586a1 1 0 01.293-.707l5.964-5.964A6 6 0 1121 9z"></path>
							</svg>
							configure ssh
						</a>
					</div>
				{:else}
					<div class="text-center py-8">
						<div class="w-20 h-20 mx-auto mb-4 bg-[var(--nb-muted)] rounded-full flex items-center justify-center">
							<svg class="w-10 h-10 nb-text-muted" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10"></path>
							</svg>
						</div>
						<h3 class="nb-title text-xl mb-2">no environment yet</h3>
						<p class="nb-text-muted mb-6">create your personal development environment to get started</p>
						<button class="nb-button nb-button-lg nb-button-primary" on:click={() => showContainerModal = true}>
							<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6v6m0 0v6m0-6h6m-6 0H6"></path>
							</svg>
							create environment
						</button>
					</div>
				{/if}
			</div>
		</div>

		<div class="space-y-4">
			<div class="nb-card text-center">
				<div class="w-12 h-12 mx-auto mb-3 bg-[var(--nb-info)] rounded-lg flex items-center justify-center">
					<svg class="w-6 h-6 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 12a9 9 0 01-9 9m9-9a9 9 0 00-9-9m9 9H3m9 9a9 9 0 01-9-9m9 9c1.657 0 3-4.03 3-9s-1.343-9-3-9m0 18c-1.657 0-3-4.03-3-9s1.343-9 3-9m-9 9a9 9 0 019-9"></path>
					</svg>
				</div>
				<div class="nb-title text-2xl">{subdomains?.length || 0}</div>
				<div class="nb-text-muted text-sm">subdomains</div>
			</div>
			
			<div class="nb-card text-center">
				<div class="w-12 h-12 mx-auto mb-3 bg-[var(--nb-secondary)] rounded-lg flex items-center justify-center">
					<svg class="w-6 h-6 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 9l3 3-3 3m5 0h3M5 20h14a2 2 0 002-2V6a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z"></path>
					</svg>
				</div>
				<div class="nb-title text-2xl">{container?.AllocatedPorts?.length || 0}</div>
				<div class="nb-text-muted text-sm">ports</div>
			</div>
		</div>
	</div>

	<div class="nb-card-lg">
		<div class="flex items-center justify-between mb-6">
			<h2 class="nb-title text-2xl">subdomains</h2>
			{#if container}
				<button class="nb-button nb-button-primary" on:click={() => showSubdomainModal = true}>
					<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6v6m0 0v6m0-6h6m-6 0H6"></path>
					</svg>
					add subdomain
				</button>
			{/if}
		</div>
		
		{#if subdomains?.length}
			<div class="grid gap-4">
				{#each subdomains as subdomain}
					<div class="nb-card flex items-center justify-between">
						<div class="flex items-center gap-4">
							<div class="w-10 h-10 bg-[var(--nb-primary)] rounded-lg flex items-center justify-center">
								<svg class="w-5 h-5 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 12a9 9 0 01-9 9m9-9a9 9 0 00-9-9m9 9H3m9 9a9 9 0 01-9-9m9 9c1.657 0 3-4.03 3-9s-1.343-9-3-9m0 18c-1.657 0-3-4.03-3-9s1.343-9 3-9m-9 9a9 9 0 019-9"></path>
								</svg>
							</div>
							<div>
								<div class="nb-mono font-bold">
									{#if subdomain.SubdomainType === 'username'}
										{subdomain.Subdomain}.hack.kim
									{:else}
										{subdomain.Subdomain}.{user.Username}.hack.kim
									{/if}
								</div>
								<div class="text-sm nb-text-muted">→ port {subdomain.TargetPort}</div>
							</div>
						</div>
						<div class="flex items-center gap-2">
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
				{/each}
			</div>
		{:else}
			<div class="text-center py-8">
				<div class="w-16 h-16 mx-auto mb-4 bg-[var(--nb-muted)] rounded-full flex items-center justify-center">
					<svg class="w-8 h-8 nb-text-muted" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 12a9 9 0 01-9 9m9-9a9 9 0 00-9-9m9 9H3m9 9a9 9 0 01-9-9m9 9c1.657 0 3-4.03 3-9s-1.343-9-3-9m0 18c-1.657 0-3-4.03-3-9s1.343-9 3-9m-9 9a9 9 0 019-9"></path>
					</svg>
				</div>
				<h3 class="nb-title text-lg mb-2">no subdomains yet</h3>
				<p class="nb-text-muted mb-4">create subdomains to expose your applications to the internet</p>
				{#if !container}
					<p class="text-sm nb-text-muted">create an environment first to manage subdomains</p>
				{/if}
			</div>
		{/if}
	</div>
</main>

<Modal 
	show={showContainerModal} 
	title="create environment" 
	onClose={() => showContainerModal = false}
>
	<div class="text-center py-4">
		<div class="w-16 h-16 mx-auto mb-4 bg-[var(--nb-primary)] rounded-full flex items-center justify-center">
			<svg class="w-8 h-8 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10"></path>
			</svg>
		</div>
		<h3 class="nb-title text-xl mb-2">create your environment</h3>
		<p class="nb-text-muted mb-6">this will create your personal development container. it may take a few minutes.</p>
	</div>
	
	<div slot="footer">
		<button class="nb-button nb-button-secondary" on:click={() => showContainerModal = false}>cancel</button>
		<button class="nb-button nb-button-primary" on:click={createContainer}>create environment</button>
	</div>
</Modal>

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

