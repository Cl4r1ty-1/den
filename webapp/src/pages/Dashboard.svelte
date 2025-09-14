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
	let containerCreating = false
	let creationProgress = 0
	let stats = null
	let statsTimer = null

	$: if (newSubdomain.subdomain_type === 'username') {
		newSubdomain.subdomain = user?.username || ''
	}

	async function createContainer() {
		containerCreating = true
		creationProgress = 0
		showContainerModal = false
		
		try {
			toastContainer.addToast('Creating your environment...', 'info')
			creationProgress = 20
			
		const res = await fetch('/user/container/create', { method: 'POST', headers: { 'Content-Type': 'application/json' } })
			creationProgress = 40
			
		const data = await res.json()
			creationProgress = 60
			
		if (data.error) {
				containerCreating = false
				creationProgress = 0
				toastContainer.addToast(data.error, 'danger')
			return
			}
			
			creationProgress = 80
			toastContainer.addToast('Environment creation started! This may take a few minutes...', 'success')
			
			let checkAttempts = 0
			const maxAttempts = 30
			
			const checkStatus = async () => {
				try {
					const statusRes = await fetch('/user/container')
					const statusData = await statusRes.json()
					
					checkAttempts++
					creationProgress = Math.min(80 + (checkAttempts * 2), 98)
					
					if (statusData.container && statusData.container.status === 'running') {
						creationProgress = 100
						containerCreating = false
						toastContainer.addToast('Environment ready! Reloading page...', 'success')
						setTimeout(() => location.reload(), 1500)
					} else if (checkAttempts < maxAttempts) {
						setTimeout(checkStatus, 10000)
					} else {
						containerCreating = false
						creationProgress = 0
						toastContainer.addToast('Environment is taking longer than expected. Please refresh the page in a few minutes.', 'warning')
					}
				} catch (error) {
					console.error('Error checking container status:', error)
					if (checkAttempts < maxAttempts) {
						setTimeout(checkStatus, 10000)
					}
				}
			}
			
			setTimeout(checkStatus, 5000)
			
		} catch (error) {
			console.error('Error creating container:', error)
			containerCreating = false
			creationProgress = 0
			toastContainer.addToast('Failed to create container: ' + error.message, 'danger')
		}
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
		setTimeout(() => location.reload(), 1000)
	}

	async function deleteSubdomain(id) {
		if (!confirm('Are you sure you want to delete this subdomain?')) return
		
		const res = await fetch(`/user/subdomains/${id}`, { method: 'DELETE' })
		const data = await res.json()
		if (data.error) {
			toastContainer.addToast(data.error, 'danger')
			return
		}
		toastContainer.addToast('Subdomain deleted successfully!', 'success')
		setTimeout(() => location.reload(), 1000)
	}

	async function pollStats() {
		if (!container) return
		try {
			const res = await fetch('/user/container/stats')
			if (!res.ok) return
			stats = await res.json()
		} catch (e) {
		}
	}

	$: if (container) {
		clearInterval(statsTimer)
		pollStats()
		statsTimer = setInterval(pollStats, 5000)
	}
</script>

<div class="min-h-screen bg-background text-foreground">
	<Header {user} currentPage="dashboard" />
	
	<main class="max-w-6xl mx-auto p-6">
		<div class="mb-8">
			<h1 class="text-4xl font-heading mb-2">welcome back, {user.display_name} 👋</h1>
			<p class="text-foreground/70">manage your cozy *nix environment</p>
		</div>

		<div class="grid md:grid-cols-3 gap-6 mb-8">
			<div class="bg-secondary-background border-2 border-border p-6 text-center shadow-shadow">
				<div class="w-12 h-12 mx-auto mb-3 bg-chart-1 border-2 border-border flex items-center justify-center">
					<svg class="w-6 h-6 text-main-foreground" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10"></path>
					</svg>
				</div>
				<div class="text-2xl font-heading">{container ? 'running' : 'no environment'}</div>
				<div class="text-foreground/70 text-sm">status</div>
			</div>
			
			<div class="bg-secondary-background border-2 border-border p-6 text-center shadow-shadow">
				<div class="w-12 h-12 mx-auto mb-3 bg-chart-2 border-2 border-border flex items-center justify-center">
					<svg class="w-6 h-6 text-main-foreground" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 12a9 9 0 01-9 9m9-9a9 9 0 00-9-9m9 9H3m9 9a9 9 0 01-9-9m9 9c1.657 0 3-4.03 3-9s-1.343-9-3-9m0 18c-1.657 0-3-4.03-3-9s1.343-9 3-9m-9 9a9 9 0 019-9"></path>
					</svg>
				</div>
				<div class="text-2xl font-heading">{subdomains?.length || 0}</div>
				<div class="text-foreground/70 text-sm">subdomains</div>
			</div>
			
			<div class="bg-secondary-background border-2 border-border p-6 text-center shadow-shadow">
				<div class="w-12 h-12 mx-auto mb-3 bg-chart-3 border-2 border-border flex items-center justify-center">
					<svg class="w-6 h-6 text-main-foreground" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 9l3 3-3 3m5 0h3M5 20h14a2 2 0 002-2V6a2 2 0 00-2-2H5a2 2 0 00-2 2v14a2 2 0 002 2z"></path>
					</svg>
				</div>
				<div class="text-2xl font-heading">{container?.allocated_ports?.length || 0}</div>
				<div class="text-foreground/70 text-sm">ports</div>
			</div>
		</div>
		
		{#if containerCreating}
			<div class="bg-secondary-background border-2 border-border p-6 shadow-shadow mb-8">
				<div class="flex items-center justify-between mb-4">
					<h2 class="text-xl font-heading">creating environment...</h2>
					<div class="text-sm text-foreground/70">{creationProgress}%</div>
				</div>
				<div class="w-full bg-background border-2 border-border h-4 overflow-hidden">
					<div 
						class="h-full bg-chart-4 transition-all duration-500 ease-out"
						style="width: {creationProgress}%"
					></div>
				</div>
				<p class="text-sm text-foreground/70 mt-2">
					Please wait while we set up your development environment. This usually takes 2-5 minutes.
				</p>
			</div>
		{/if}
		
		<div class="grid lg:grid-cols-2 gap-8 mb-8">
			<div class="lg:col-span-2">
				<div class="bg-secondary-background border-2 border-border p-6 shadow-shadow">
					<div class="flex items-center justify-between mb-6">
						<h2 class="text-2xl font-heading">environment status</h2>
						{#if container}
							<div class="bg-chart-4 text-main-foreground px-3 py-1 border-2 border-border text-sm font-heading">
								<svg class="w-4 h-4 inline mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"></path>
								</svg>
								running
							</div>
						{:else}
							<div class="bg-chart-1 text-main-foreground px-3 py-1 border-2 border-border text-sm font-heading">
								<svg class="w-4 h-4 inline mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2m7-2a9 9 0 11-18 0 9 9 0 0118 0z"></path>
								</svg>
								no environment
							</div>
						{/if}
					</div>
					
					{#if container}
						<div class="grid md:grid-cols-2 gap-6">
							<div>
								<div class="flex items-center justify-between mb-3">
									<h3 class="font-heading">SSH Access</h3>
									<a 
										href="/user/ssh-setup"
										class="bg-chart-2 text-main-foreground border-2 border-border px-3 py-1 text-sm font-heading hover:translate-x-1 hover:translate-y-1 transition-transform shadow-shadow"
									>
										<svg class="w-4 h-4 inline mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"></path>
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"></path>
									</svg>
									setup ssh
									</a>
								</div>
								<div class="bg-background border-2 border-border p-4 font-mono text-sm">
									ssh {user.username}@hack.kim
								</div>
								<p class="text-foreground/70 text-sm mt-2">
									Use this command to connect to your environment. Set up SSH keys or password first.
								</p>
							</div>
							
							<div>
								<div class="flex items-center justify-between mb-3">
									<h3 class="font-heading">Allocated Ports</h3>
									<button 
										class="bg-main text-main-foreground border-2 border-border px-3 py-1 text-sm font-heading hover:translate-x-1 hover:translate-y-1 transition-transform shadow-shadow"
										on:click={getNewPort}
									>
										<svg class="w-4 h-4 inline mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6v6m0 0v6m0-6h6m-6 0H6"></path>
										</svg>
										get new port
									</button>
								</div>
								{#if container?.allocated_ports?.length}
									<div class="flex flex-wrap gap-2">
										{#each container.allocated_ports as port}
											<div class="bg-background border-2 border-border px-2 py-1 text-sm font-mono">{port}</div>
					{/each}
									</div>
								{:else}
									<p class="text-foreground/70 text-sm">No ports allocated yet</p>
								{/if}
							</div>
							<div>
								<h3 class="font-heading mb-3">Live Stats</h3>
								{#if stats}
									<div class="grid grid-cols-2 gap-3">
										<div class="bg-background border-2 border-border p-3">
											<div class="text-sm text-foreground/70">CPU</div>
											<div class="font-mono">{Math.round((stats.cpu_usage_ns || 0) / 1e7) / 100}%</div>
										</div>
										<div class="bg-background border-2 border-border p-3">
											<div class="text-sm text-foreground/70">Memory</div>
											<div class="font-mono">{Math.round(((stats.memory_usage_bytes||0) / 1024 / 1024) * 10) / 10} MB</div>
										</div>
										<div class="bg-background border-2 border-border p-3">
											<div class="text-sm text-foreground/70">Disk</div>
											<div class="font-mono">{Math.round(((stats.disk_usage_bytes||0) / 1024 / 1024) * 10) / 10} MB</div>
										</div>
										<div class="bg-background border-2 border-border p-3">
											<div class="text-sm text-foreground/70">Net RX / TX</div>
											<div class="font-mono">{Math.round(((stats.network_rx_bytes||0) / 1024 / 1024) * 10) / 10} / {Math.round(((stats.network_tx_bytes||0) / 1024 / 1024) * 10) / 10} MB</div>
										</div>
									</div>
								{:else}
									<p class="text-foreground/70 text-sm">collecting stats...</p>
								{/if}
							</div>
						</div>
			{:else}
						<div class="text-center py-12">
							<div class="w-20 h-20 mx-auto mb-6 bg-foreground/10 border-2 border-border flex items-center justify-center">
								<svg class="w-10 h-10 text-foreground/50" fill="none" stroke="currentColor" viewBox="0 0 24 24">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10"></path>
								</svg>
							</div>
							<h3 class="text-xl font-heading mb-2">no environment yet</h3>
							<p class="text-foreground/70 mb-6">create your personal development environment to get started</p>
							<button 
								class="bg-main text-main-foreground border-2 border-border px-6 py-3 text-lg font-heading hover:translate-x-1 hover:translate-y-1 transition-transform shadow-shadow disabled:opacity-50 disabled:cursor-not-allowed disabled:transform-none"
								disabled={containerCreating}
								on:click={() => showContainerModal = true}
							>
								<svg class="w-5 h-5 inline mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6v6m0 0v6m0-6h6m-6 0H6"></path>
								</svg>
								create environment
							</button>
						</div>
			{/if}
		</div>
	</div>
</div>

		<div class="bg-secondary-background border-2 border-border p-6 shadow-shadow">
			<div class="flex items-center justify-between mb-6">
				<h2 class="text-2xl font-heading">subdomains</h2>
				{#if container}
					<button 
						class="bg-main text-main-foreground border-2 border-border px-4 py-2 font-heading hover:translate-x-1 hover:translate-y-1 transition-transform shadow-shadow"
						on:click={() => showSubdomainModal = true}
					>
						<svg class="w-4 h-4 inline mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6v6m0 0v6m0-6h6m-6 0H6"></path>
						</svg>
						add subdomain
					</button>
				{/if}
			</div>
			
			{#if subdomains?.length}
				<div class="grid gap-4">
					{#each subdomains as subdomain}
						<div class="bg-background border-2 border-border p-4 flex items-center justify-between shadow-shadow">
							<div class="flex items-center gap-4">
								<div class="w-10 h-10 bg-chart-2 border-2 border-border flex items-center justify-center">
									<svg class="w-5 h-5 text-main-foreground" fill="none" stroke="currentColor" viewBox="0 0 24 24">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 12a9 9 0 01-9 9m9-9a9 9 0 00-9-9m9 9H3m9 9a9 9 0 01-9-9m9 9c1.657 0 3-4.03 3-9s-1.343-9-3-9m0 18c-1.657 0-3-4.03-3-9s1.343-9 3-9m-9 9a9 9 0 019-9"></path>
									</svg>
					</div>
					<div>
									<div class="font-mono font-bold">
										{#if subdomain.subdomain_type === 'username'}
											{subdomain.subdomain}.hack.kim
										{:else}
											{subdomain.subdomain}.{user.username}.hack.kim
										{/if}
									</div>
									<div class="text-sm text-foreground/70">→ port {subdomain.target_port}</div>
								</div>
							</div>
							<div class="flex items-center gap-2">
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
								<button 
									class="bg-chart-1 text-main-foreground border-2 border-border px-3 py-1 text-sm font-heading hover:translate-x-1 hover:translate-y-1 transition-transform shadow-shadow"
									on:click={() => deleteSubdomain(subdomain.id)}
								>
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
					<div class="w-16 h-16 mx-auto mb-4 bg-foreground/10 border-2 border-border flex items-center justify-center">
						<svg class="w-8 h-8 text-foreground/50" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 12a9 9 0 01-9 9m9-9a9 9 0 00-9-9m9 9H3m9 9a9 9 0 01-9-9m9 9c1.657 0 3-4.03 3-9s-1.343-9-3-9m0 18c-1.657 0-3-4.03-3-9s1.343-9 3-9m-9 9a9 9 0 019-9"></path>
						</svg>
					</div>
					<h3 class="text-lg font-heading mb-2">no subdomains yet</h3>
					<p class="text-foreground/70 mb-4">create subdomains to expose your applications to the internet</p>
					{#if !container}
						<p class="text-sm text-foreground/70">create an environment first to manage subdomains</p>
					{/if}
				</div>
			{/if}
		</div>
	</main>
				</div>

<Modal show={showContainerModal} title="Create Environment" onClose={() => showContainerModal = false}>
	<div class="space-y-4">
		<p class="text-foreground/70">
			This will create a new development environment for you. It may take a few minutes to set up.
		</p>
		<div class="bg-background border-2 border-border p-4">
			<h4 class="font-heading mb-2">What you'll get:</h4>
			<ul class="text-sm text-foreground/70 space-y-1">
				<li>• Ubuntu-based container</li>
				<li>• SSH access</li>
				<li>• 4GB RAM, 4 CPU cores</li>
				<li>• 15GB storage</li>
				<li>• Network ports for your applications</li>
			</ul>
		</div>
	</div>
	
	<div slot="footer" class="flex gap-3">
		<button 
			class="bg-foreground/10 border-2 border-border px-4 py-2 font-heading hover:translate-x-1 hover:translate-y-1 transition-transform"
			on:click={() => showContainerModal = false}
		>
			cancel
		</button>
		<button 
			class="bg-main text-main-foreground border-2 border-border px-4 py-2 font-heading hover:translate-x-1 hover:translate-y-1 transition-transform shadow-shadow"
			on:click={createContainer}
		>
			create environment
		</button>
	</div>
</Modal>

<Modal show={showSubdomainModal} title="Add Subdomain" onClose={() => showSubdomainModal = false}>
	<form on:submit|preventDefault={createSubdomain} class="space-y-4">
		<div>
			<label class="block text-sm font-heading mb-2">subdomain type</label>
			<div class="space-y-2">
				<label class="flex items-center gap-2 cursor-pointer">
					<input type="radio" bind:group={newSubdomain.subdomain_type} value="project" class="w-4 h-4">
					<span>project subdomain (myapp.{user.username}.hack.kim)</span>
				</label>
				<label class="flex items-center gap-2 cursor-pointer">
					<input type="radio" bind:group={newSubdomain.subdomain_type} value="username" class="w-4 h-4">
					<span>username subdomain ({user.username}.hack.kim)</span>
				</label>
			</div>
		</div>
		
		{#if newSubdomain.subdomain_type === 'username'}
			<div>
				<label class="block text-sm font-heading mb-2">domain</label>
				<div class="w-full bg-background border-2 border-border p-3">
					your project will be on <span class="font-mono font-bold">{user.username}.hack.kim</span>
				</div>
			</div>
		{:else}
			<div>
				<label class="block text-sm font-heading mb-2">subdomain name</label>
				<input 
					type="text" 
					bind:value={newSubdomain.subdomain} 
					required={newSubdomain.subdomain_type !== 'username'} 
					class="w-full bg-background border-2 border-border p-3 font-mono"
					placeholder="myapp"
				>
				<div class="text-xs text-foreground/70 mt-1">
					preview: {newSubdomain.subdomain || 'myapp'}.{user.username}.hack.kim
		</div>
	</div>
{/if}

		<div>
			<label class="block text-sm font-heading mb-2">target port</label>
			<select bind:value={newSubdomain.target_port} required class="w-full bg-background border-2 border-border p-3">
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
		<button 
			class="bg-foreground/10 border-2 border-border px-4 py-2 font-heading hover:translate-x-1 hover:translate-y-1 transition-transform"
			on:click={() => showSubdomainModal = false}
		>
			cancel
		</button>
		<button 
			class="bg-main text-main-foreground border-2 border-border px-4 py-2 font-heading hover:translate-x-1 hover:translate-y-1 transition-transform shadow-shadow"
			on:click={async () => {
				if (newSubdomain.subdomain_type === 'username') {
					newSubdomain.subdomain = user.username
				}
				await createSubdomain()
			}}
		>
			create subdomain
		</button>
	</div>
</Modal>


<ToastContainer bind:this={toastContainer} />