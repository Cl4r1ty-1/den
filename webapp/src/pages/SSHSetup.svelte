<script>
	import Header from '../lib/Header.svelte'
	import ToastContainer from '../lib/ToastContainer.svelte'
	
	export let user
	
	let method = 'password'
	let password = ''
	let confirmPassword = ''
	let publicKey = ''
	let toastContainer
	
	async function savePassword() {
		if (!password) {
			toastContainer.addToast('Please enter a password', 'danger')
			return
		}
		
		if (password !== confirmPassword) {
			toastContainer.addToast('Passwords do not match', 'danger')
			return
		}
		
		if (password.length < 8) {
			toastContainer.addToast('Password must be at least 8 characters long', 'danger')
			return
		}
		
		const res = await fetch('/user/ssh-setup', {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ method: 'password', password })
		})
		
		const data = await res.json()
		if (data.error) {
			toastContainer.addToast(data.error, 'danger')
			return
		}
		
		toastContainer.addToast('Password set successfully! You can now SSH into your environment.', 'success')
	}
	
	async function savePublicKey() {
		if (!publicKey.trim()) {
			toastContainer.addToast('Please enter your SSH public key', 'danger')
			return
		}
		
		if (!publicKey.startsWith('ssh-rsa') && !publicKey.startsWith('ssh-ed25519')) {
			toastContainer.addToast('Please enter a valid SSH public key (should start with ssh-rsa or ssh-ed25519)', 'danger')
			return
		}
		
		const res = await fetch('/user/ssh-setup', {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ method: 'key', public_key: publicKey })
		})
		
		const data = await res.json()
		if (data.error) {
			toastContainer.addToast(data.error, 'danger')
			return
		}
		
		toastContainer.addToast('Public key set successfully! You can now SSH into your environment.', 'success')
	}
</script>

<div class="min-h-screen bg-background text-foreground">
	<Header {user} currentPage="ssh-setup" />
	
	<main class="max-w-4xl mx-auto p-6">
		<div class="mb-8">
			<a href="/user/dashboard" class="inline-flex items-center gap-2 bg-foreground/10 border-2 border-border px-3 py-2 text-sm font-heading hover:translate-x-1 hover:translate-y-1 transition-transform mb-4">
				<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7"></path>
				</svg>
				back to dashboard
			</a>
			<h1 class="text-4xl font-heading mb-2">ssh configuration</h1>
			<p class="text-xl text-foreground/70">configure secure access to your environment</p>
		</div>

		<div class="bg-secondary-background border-2 border-border p-6 shadow-shadow mb-8">
			<h2 class="text-2xl font-heading mb-6">authentication method</h2>
			
			<div class="grid md:grid-cols-2 gap-6 mb-8">
				<button 
					class="bg-secondary-background border-2 border-border p-6 shadow-shadow cursor-pointer transition-all duration-200 hover:translate-x-1 hover:translate-y-1 text-left {method === 'password' ? 'border-chart-3' : ''}"
					on:click={() => method = 'password'}
				>
					<div class="flex items-center gap-4 mb-4">
						<div class="w-12 h-12 bg-chart-3 border-2 border-border flex items-center justify-center">
							<svg class="w-6 h-6 text-main-foreground" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z"></path>
							</svg>
						</div>
						<div>
							<h3 class="text-lg font-heading">password authentication</h3>
							<p class="text-chart-4 text-sm font-heading">quick and easy setup</p>
						</div>
					</div>
					<p class="text-foreground/70 text-sm">
						Use a password to authenticate. Simple but less secure than public key authentication.
					</p>
				</button>
				
				<button 
					class="bg-secondary-background border-2 border-border p-6 shadow-shadow cursor-pointer transition-all duration-200 hover:translate-x-1 hover:translate-y-1 text-left {method === 'key' ? 'border-chart-4' : ''}"
					on:click={() => method = 'key'}
				>
					<div class="flex items-center gap-4 mb-4">
						<div class="w-12 h-12 bg-chart-4 border-2 border-border flex items-center justify-center">
							<svg class="w-6 h-6 text-main-foreground" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 7a2 2 0 012 2m4 0a6 6 0 01-7.743 5.743L11 17H9v2H7v2H4a1 1 0 01-1-1v-2.586a1 1 0 01.293-.707l5.964-5.964A6 6 0 1721 9z"></path>
							</svg>
						</div>
						<div>
							<h3 class="text-lg font-heading">public key authentication</h3>
							<p class="text-chart-4 text-sm font-heading">recommended for security</p>
						</div>
					</div>
					<p class="text-foreground/70 text-sm">
						Use SSH keys for secure, passwordless authentication. More secure and convenient.
					</p>
				</button>
			</div>
		</div>

		{#if method === 'password'}
			<div class="bg-secondary-background border-2 border-border p-6 shadow-shadow">
				<h2 class="text-xl font-heading mb-6">password configuration</h2>
				
				<div class="bg-chart-3 border-2 border-border p-4 mb-6">
					<div class="flex items-start gap-3">
						<svg class="w-5 h-5 mt-0.5 text-main-foreground" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.964-.833-2.732 0L3.732 16.5c-.77.833.192 2.5 1.732 2.5z"></path>
						</svg>
						<div>
							<h4 class="font-heading mb-1 text-main-foreground">security note</h4>
							<p class="text-sm text-main-foreground">Use a strong, unique password. Consider using public key authentication for better security.</p>
						</div>
					</div>
				</div>
				
				<form on:submit|preventDefault={savePassword} class="space-y-6">
					<div>
						<label class="block text-sm font-heading mb-2" for="ssh_password">password</label>
						<input id="ssh_password" type="password" bind:value={password} class="w-full bg-background border-2 border-border p-3">
					</div>
					
					<div>
						<label class="block text-sm font-heading mb-2" for="ssh_password_confirm">confirm password</label>
						<input id="ssh_password_confirm" type="password" bind:value={confirmPassword} class="w-full bg-background border-2 border-border p-3">
					</div>
					
					<button type="submit" class="bg-main text-main-foreground border-2 border-border px-6 py-3 font-heading hover:translate-x-1 hover:translate-y-1 transition-transform shadow-shadow flex items-center gap-2">
						<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"></path>
						</svg>
						set password
					</button>
				</form>
			</div>
		{:else}
			<div class="bg-secondary-background border-2 border-border p-6 shadow-shadow">
				<h2 class="text-xl font-heading mb-6">public key configuration</h2>
				
				<div class="bg-chart-2 border-2 border-border p-4 mb-6">
					<div class="flex items-start gap-3">
						<svg class="w-5 h-5 mt-0.5 text-main-foreground" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path>
						</svg>
						<div>
							<h4 class="font-heading mb-2 text-main-foreground">generate ssh key pair</h4>
							<p class="text-sm mb-3 text-main-foreground">if you don't have an SSH key pair, generate one with:</p>
							<div class="bg-background border-2 border-border p-3 font-mono text-sm">
								ssh-keygen -t ed25519 -C "your@email.com"
							</div>
							<p class="text-sm mt-2 text-main-foreground">then copy the contents of <code class="font-mono bg-background px-1 border border-border">~/.ssh/id_ed25519.pub</code></p>
						</div>
					</div>
				</div>
				
				<form on:submit|preventDefault={savePublicKey} class="space-y-6">
					<div>
						<label class="block text-sm font-heading mb-2" for="ssh_public_key">ssh public key</label>
						<textarea id="ssh_public_key" bind:value={publicKey} class="w-full bg-background border-2 border-border p-3 font-mono" rows="6"></textarea>
						<p class="text-sm text-foreground/70 mt-2">
							paste your public key here. it should start with <code class="font-mono">ssh-ed25519</code> or <code class="font-mono">ssh-rsa</code>
						</p>
					</div>
					
					<button type="submit" class="bg-main text-main-foreground border-2 border-border px-6 py-3 font-heading hover:translate-x-1 hover:translate-y-1 transition-transform shadow-shadow flex items-center gap-2">
						<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"></path>
						</svg>
						set public key
					</button>
				</form>
			</div>
		{/if}

		<div class="bg-secondary-background border-2 border-border p-6 shadow-shadow mt-8">
			<h2 class="text-xl font-heading mb-4">connection information</h2>
			<div class="bg-background border-2 border-border p-4 mb-4">
				<h3 class="font-heading mb-2">ssh command</h3>
				<div class="bg-background border-2 border-border p-3 font-mono text-sm">
					ssh {user.username}@hack.kim
				</div>
			</div>
			
			<div class="grid md:grid-cols-2 gap-4 text-sm">
				<div>
					<h4 class="font-heading mb-2 text-foreground/70">what you get:</h4>
					<ul class="space-y-1 text-foreground/70">
						<li>• full shell access</li>
						<li>• persistent home directory</li>
						<li>• pre-installed development tools</li>
						<li>• ability to install packages</li>
					</ul>
				</div>
				
				<div>
					<h4 class="font-heading mb-2 text-foreground/70">tips:</h4>
					<ul class="space-y-1 text-foreground/70">
						<li>• use <code class="font-mono">tmux</code> for persistent sessions</li>
						<li>• your files are automatically backed up</li>
						<li>• use ports from your allocated range</li>
						<li>• check <code class="font-mono">~/README</code> for more info</li>
					</ul>
				</div>
			</div>
		</div>
	</main>
</div>

<ToastContainer bind:this={toastContainer} />
