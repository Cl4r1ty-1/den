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

<Header {user} />

<main class="nb-container py-8">
	<div class="max-w-4xl mx-auto">
		<div class="mb-8">
			<a href="/user/dashboard" class="nb-button nb-button-sm mb-4">
				<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7"></path>
				</svg>
				back to dashboard
			</a>
			<h1 class="nb-title text-4xl mb-2">ssh configuration</h1>
			<p class="nb-subtitle text-xl">configure secure access to your environment</p>
		</div>

		<div class="nb-card-lg mb-8">
			<h2 class="nb-title text-2xl mb-6">authentication method</h2>
			
			<div class="grid md:grid-cols-2 gap-6 mb-8">
				<div 
					class="nb-card cursor-pointer transition-all duration-200 {method === 'password' ? 'ring-4 ring-[var(--nb-primary)] ring-opacity-50' : ''}"
					on:click={() => method = 'password'}
					role="button"
					tabindex="0"
				>
					<div class="flex items-center gap-4 mb-4">
						<div class="w-12 h-12 bg-[var(--nb-warning)] rounded-lg flex items-center justify-center">
							<svg class="w-6 h-6 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z"></path>
							</svg>
						</div>
						<div>
							<h3 class="nb-title text-lg">password authentication</h3>
							<p class="nb-text-muted text-sm">quick and easy setup</p>
						</div>
					</div>
					<p class="text-sm nb-text-muted">
						Use a password to authenticate. Simple but less secure than public key authentication.
					</p>
				</div>
				
				<div 
					class="nb-card cursor-pointer transition-all duration-200 {method === 'key' ? 'ring-4 ring-[var(--nb-primary)] ring-opacity-50' : ''}"
					on:click={() => method = 'key'}
					role="button"
					tabindex="0"
				>
					<div class="flex items-center gap-4 mb-4">
						<div class="w-12 h-12 bg-[var(--nb-success)] rounded-lg flex items-center justify-center">
							<svg class="w-6 h-6 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 7a2 2 0 012 2m4 0a6 6 0 01-7.743 5.743L11 17H9v2H7v2H4a1 1 0 01-1-1v-2.586a1 1 0 01.293-.707l5.964-5.964A6 6 0 1721 9z"></path>
							</svg>
						</div>
						<div>
							<h3 class="nb-title text-lg">public key authentication</h3>
							<p class="nb-text-muted text-sm">recommended for security</p>
						</div>
					</div>
					<p class="text-sm nb-text-muted">
						Use SSH keys for secure, passwordless authentication. More secure and convenient.
					</p>
				</div>
			</div>
		</div>

		{#if method === 'password'}
			<div class="nb-card-lg">
				<h2 class="nb-title text-xl mb-6">password configuration</h2>
				
				<div class="nb-card bg-[var(--nb-warning)] text-[var(--nb-accent)] mb-6">
					<div class="flex items-start gap-3">
						<svg class="w-5 h-5 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.964-.833-2.732 0L3.732 16.5c-.77.833.192 2.5 1.732 2.5z"></path>
						</svg>
						<div>
							<h4 class="font-bold mb-1">security note</h4>
							<p class="text-sm">Use a strong, unique password. Consider using public key authentication for better security.</p>
						</div>
					</div>
				</div>
				
				<form on:submit|preventDefault={savePassword} class="space-y-6">
					<div>
						<label class="nb-label">password</label>
						<input 
							type="password" 
							bind:value={password}
							placeholder="enter a strong password"
							class="nb-input"
							required
						>
					</div>
					
					<div>
						<label class="nb-label">confirm password</label>
						<input 
							type="password" 
							bind:value={confirmPassword}
							placeholder="confirm your password"
							class="nb-input"
							required
						>
					</div>
					
					<button type="submit" class="nb-button nb-button-lg nb-button-primary">
						<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"></path>
						</svg>
						set password
					</button>
				</form>
			</div>
		{:else}
			<div class="nb-card-lg">
				<h2 class="nb-title text-xl mb-6">public key configuration</h2>
				
				<div class="nb-card bg-[var(--nb-info)] text-white mb-6">
					<div class="flex items-start gap-3">
						<svg class="w-5 h-5 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path>
						</svg>
						<div>
							<h4 class="font-bold mb-2">generate ssh key pair</h4>
							<p class="text-sm mb-3">if you don't have an SSH key pair, generate one with:</p>
							<div class="nb-card bg-[var(--nb-accent)] text-[var(--nb-success)] nb-mono text-sm p-3">
								ssh-keygen -t ed25519 -C "your@email.com"
							</div>
							<p class="text-sm mt-2">then copy the contents of <code class="nb-mono bg-white bg-opacity-20 px-1 rounded">~/.ssh/id_ed25519.pub</code></p>
						</div>
					</div>
				</div>
				
				<form on:submit|preventDefault={savePublicKey} class="space-y-6">
					<div>
						<label class="nb-label">ssh public key</label>
						<textarea 
							bind:value={publicKey}
							placeholder="ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAI... your@email.com"
							class="nb-input nb-textarea"
							rows="4"
							required
						></textarea>
						<p class="text-sm nb-text-muted mt-2">
							paste your public key here. it should start with <code class="nb-mono">ssh-ed25519</code> or <code class="nb-mono">ssh-rsa</code>
						</p>
					</div>
					
					<button type="submit" class="nb-button nb-button-lg nb-button-primary">
						<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"></path>
						</svg>
						set public key
					</button>
				</form>
			</div>
		{/if}

		<div class="nb-card-lg mt-8">
			<h2 class="nb-title text-xl mb-4">connection information</h2>
			<div class="nb-card bg-[var(--nb-surface-alt)] mb-4">
				<h3 class="font-bold mb-2">ssh command</h3>
				<div class="nb-card bg-[var(--nb-accent)] text-[var(--nb-success)] nb-mono text-sm p-3">
					ssh {user.Username}@hack.kim
				</div>
			</div>
			
			<div class="grid md:grid-cols-2 gap-4 text-sm">
				<div>
					<h4 class="font-bold mb-2 nb-text-muted">what you get:</h4>
					<ul class="space-y-1 nb-text-muted">
						<li>• full shell access</li>
						<li>• persistent home directory</li>
						<li>• pre-installed development tools</li>
						<li>• ability to install packages</li>
					</ul>
				</div>
				
				<div>
					<h4 class="font-bold mb-2 nb-text-muted">tips:</h4>
					<ul class="space-y-1 nb-text-muted">
						<li>• use <code class="nb-mono">tmux</code> for persistent sessions</li>
						<li>• your files are automatically backed up</li>
						<li>• use ports from your allocated range</li>
						<li>• check <code class="nb-mono">~/README</code> for more info</li>
					</ul>
				</div>
			</div>
		</div>
	</div>
</main>

<ToastContainer bind:this={toastContainer} />
