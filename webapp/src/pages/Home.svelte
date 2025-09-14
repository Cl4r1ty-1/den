<script>
	function login() { 
		window.location.href = '/auth/github' 
	}
	export let user = null
</script>

<div class="min-h-screen bg-background text-foreground">
	<main class="flex items-center justify-center min-h-screen p-6">
		<div class="max-w-4xl text-center">
			<div class="mb-12">
				<h1 class="text-8xl md:text-9xl font-heading mb-6">
					<span class="text-chart-1">d</span><span class="text-chart-4">e</span><span class="text-chart-2">n</span>
				</h1>
				<p class="text-3xl md:text-4xl font-heading mb-4">your cozy *nix home</p>
				<p class="text-xl text-foreground/70 max-w-2xl mx-auto mb-8">
					get your own isolated linux container with ssh access, persistent storage, 
					and all the development tools you need
				</p>
				{#if user}
					<a href="/user/dashboard" class="inline-block bg-main text-main-foreground border-2 border-border px-8 py-4 text-xl font-heading hover:translate-x-1 hover:translate-y-1 transition-transform shadow-shadow">
						go to dashboard
					</a>
				{:else}
					<button 
						class="bg-main text-main-foreground border-2 border-border px-8 py-4 text-xl font-heading hover:translate-x-1 hover:translate-y-1 transition-transform shadow-shadow"
						on:click={login}
					>
						<svg class="w-6 h-6 inline mr-3" viewBox="0 0 16 16" fill="currentColor">
							<path d="M8 0C3.58 0 0 3.58 0 8a8 8 0 0 0 5.47 7.59c.4.07.55-.17.55-.38 0-.19-.01-.82-.01-1.49-2.01.37-2.53-.49-2.69-.94-.09-.23-.48-.94-.82-1.13-.28-.15-.68-.52-.01-.53.63-.01 1.08.58 1.23.82.72 1.21 1.87.87 2.33.66.07-.52.28-.87.51-1.07-1.78-.2-3.64-.89-3.64-3.95 0-.87.31-1.59.82-2.15-.08-.2-.36-1.02.08-2.12 0 0 .67-.21 2.2.82.64-.18 1.32-.27 2-.27.68 0 1.36.09 2 .27 1.53-1.04 2.2-.82 2.2-.82.44 1.1.16 1.92.08 2.12.51.56.82 1.27.82 2.15 0 3.07-1.87 3.75-3.65 3.95.29.25.54.73.54 1.48 0 1.07-.01 1.93-.01 2.19 0 .21.15.46.55.38A8.01 8.01 0 0 0 16 8c0-4.42-3.58-8-8-8Z"/>
						</svg>
						get started with github
					</button>
				{/if}
			</div>

			<div class="grid md:grid-cols-3 gap-8 mb-12">
				<div class="bg-secondary-background border-2 border-border p-6 shadow-shadow hover:translate-x-1 hover:translate-y-1 transition-transform">
					<div class="w-16 h-16 mx-auto mb-4 bg-chart-4 border-2 border-border flex items-center justify-center">
						<svg class="w-8 h-8 text-main-foreground" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z"></path>
						</svg>
					</div>
					<h3 class="text-xl font-heading mb-2">secure & isolated</h3>
					<p class="text-foreground/70">
						each user gets their own isolated container with dedicated resources 
						and secure network isolation
					</p>
				</div>
				
				<div class="bg-secondary-background border-2 border-border p-6 shadow-shadow hover:translate-x-1 hover:translate-y-1 transition-transform">
					<div class="w-16 h-16 mx-auto mb-4 bg-chart-2 border-2 border-border flex items-center justify-center">
						<svg class="w-8 h-8 text-main-foreground" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 10V3L4 14h7v7l9-11h-7z"></path>
						</svg>
					</div>
					<h3 class="text-xl font-heading mb-2">lightning fast</h3>
					<p class="text-foreground/70">
						containers start in seconds, not minutes. 
						get coding immediately with pre-installed development tools
					</p>
				</div>
				
				<div class="bg-secondary-background border-2 border-border p-6 shadow-shadow hover:translate-x-1 hover:translate-y-1 transition-transform">
					<div class="w-16 h-16 mx-auto mb-4 bg-chart-1 border-2 border-border flex items-center justify-center">
						<svg class="w-8 h-8 text-main-foreground" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 0 0 2.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 0 0 1.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 0 0-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 0 0-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 0 0-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 0 0-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 0 0 1.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"></path>
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"></path>
						</svg>
					</div>
					<h3 class="text-xl font-heading mb-2">fully configurable</h3>
					<p class="text-foreground/70">
						install packages, configure services, and 
						customize your environment. it's your linux box in the cloud
					</p>
				</div>
			</div>

			<div class="bg-main text-main-foreground border-2 border-border p-8 shadow-shadow">
				<p class="text-lg font-heading mb-2">
					den is built for the student community. sign in with your github account to get access!
				</p>
				<p class="text-main-foreground/80">
					ssh access • persistent storage • custom subdomains • reverse proxy setup
				</p>
			</div>
		</div>
	</main>
</div>