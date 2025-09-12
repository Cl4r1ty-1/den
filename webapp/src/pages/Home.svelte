<script>
	function login() { 
		window.location.href = '/auth/github' 
	}
</script>

<div class="min-h-screen relative overflow-hidden">
	<div class="absolute inset-0 opacity-5">
		<div class="absolute inset-0" style="background-image: radial-gradient(circle at 25px 25px, var(--nb-accent) 2px, transparent 0), radial-gradient(circle at 75px 75px, var(--nb-accent) 2px, transparent 0); background-size: 100px 100px;"></div>
	</div>
	
	<div class="absolute top-20 left-10 w-20 h-20 bg-[var(--nb-primary)] rounded-lg rotate-12 opacity-20"></div>
	<div class="absolute top-40 right-20 w-16 h-16 bg-[var(--nb-secondary)] rounded-full opacity-20"></div>
	<div class="absolute bottom-40 left-20 w-24 h-24 bg-[var(--nb-success)] rounded-lg -rotate-12 opacity-20"></div>
	<div class="absolute bottom-20 right-10 w-12 h-12 bg-[var(--nb-info)] rounded-full opacity-20"></div>

	<main class="relative z-10 flex items-center justify-center min-h-screen p-6">
		<div class="nb-container max-w-6xl text-center">
			<div class="mb-16">
				<h1 class="nb-title text-8xl md:text-9xl mb-6">
					<span class="text-[var(--nb-primary)]">d</span><span class="text-[var(--nb-secondary)]">e</span><span class="text-[var(--nb-info)]">n</span>
				</h1>
				<p class="nb-subtitle text-3xl md:text-4xl mb-4">your cozy *nix home</p>
				<p class="text-xl nb-text-muted max-w-2xl mx-auto mb-8">
					get your own isolated linux container with ssh access, persistent storage, 
					and all the development tools you need
				</p>
				
				<button 
					class="nb-button nb-button-xl nb-button-primary text-xl px-12 py-4"
					on:click={login}
				>
					<svg class="w-6 h-6" viewBox="0 0 16 16" fill="currentColor">
						<path d="M8 0C3.58 0 0 3.58 0 8a8 8 0 0 0 5.47 7.59c.4.07.55-.17.55-.38 0-.19-.01-.82-.01-1.49-2.01.37-2.53-.49-2.69-.94-.09-.23-.48-.94-.82-1.13-.28-.15-.68-.52-.01-.53.63-.01 1.08.58 1.23.82.72 1.21 1.87.87 2.33.66.07-.52.28-.87.51-1.07-1.78-.2-3.64-.89-3.64-3.95 0-.87.31-1.59.82-2.15-.08-.2-.36-1.02.08-2.12 0 0 .67-.21 2.2.82.64-.18 1.32-.27 2-.27.68 0 1.36.09 2 .27 1.53-1.04 2.2-.82 2.2-.82.44 1.1.16 1.92.08 2.12.51.56.82 1.27.82 2.15 0 3.07-1.87 3.75-3.65 3.95.29.25.54.73.54 1.48 0 1.07-.01 1.93-.01 2.19 0 .21.15.46.55.38A8.01 8.01 0 0 0 16 8c0-4.42-3.58-8-8-8Z"/>
					</svg>
					get started with github
				</button>
			</div>

			<div class="grid md:grid-cols-3 gap-8 mb-16">
				<div class="nb-card-lg text-center group hover:scale-105 transition-transform duration-200">
					<div class="w-20 h-20 mx-auto mb-6 bg-[var(--nb-success)] rounded-xl flex items-center justify-center">
						<svg class="w-10 h-10 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z"></path>
						</svg>
					</div>
					<h3 class="nb-title text-2xl mb-3">secure & isolated</h3>
					<p class="nb-text-muted">
						each user gets their own isolated LXC container with dedicated resources 
						and secure network isolation
					</p>
				</div>
				
				<div class="nb-card-lg text-center group hover:scale-105 transition-transform duration-200">
					<div class="w-20 h-20 mx-auto mb-6 bg-[var(--nb-info)] rounded-xl flex items-center justify-center">
						<svg class="w-10 h-10 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 10V3L4 14h7v7l9-11h-7z"></path>
						</svg>
					</div>
					<h3 class="nb-title text-2xl mb-3">lightning fast</h3>
					<p class="nb-text-muted">
						containers start in seconds, not minutes. get coding immediately 
						with pre-installed development tools
					</p>
				</div>
				
				<div class="nb-card-lg text-center group hover:scale-105 transition-transform duration-200">
					<div class="w-20 h-20 mx-auto mb-6 bg-[var(--nb-primary)] rounded-xl flex items-center justify-center">
						<svg class="w-10 h-10 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"></path>
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"></path>
						</svg>
					</div>
					<h3 class="nb-title text-2xl mb-3">fully configurable</h3>
					<p class="nb-text-muted">
						install packages, configure services, and customize your environment. 
						it's your linux box in the cloud
					</p>
				</div>
			</div>

			<div class="mb-16">
				<h2 class="nb-title text-4xl mb-8">what you get</h2>
				<div class="grid md:grid-cols-2 gap-8 max-w-4xl mx-auto">
					<div class="nb-card text-left">
						<h3 class="nb-title text-xl mb-4 flex items-center gap-3">
							<svg class="w-6 h-6 text-[var(--nb-primary)]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 9l3 3-3 3m5 0h3M5 20h14a2 2 0 002-2V6a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z"></path>
							</svg>
							full shell access
						</h3>
						<ul class="space-y-2 nb-text-muted">
							<li>• ssh access to your container</li>
							<li>• persistent home directory</li>
							<li>• sudo privileges</li>
							<li>• pre-installed dev tools</li>
						</ul>
					</div>
					
					<div class="nb-card text-left">
						<h3 class="nb-title text-xl mb-4 flex items-center gap-3">
							<svg class="w-6 h-6 text-[var(--nb-secondary)]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 12a9 9 0 01-9 9m9-9a9 9 0 00-9-9m9 9H3m9 9a9 9 0 01-9-9m9 9c1.657 0 3-4.03 3-9s-1.343-9-3-9m0 18c-1.657 0-3-4.03-3-9s1.343-9 3-9m-9 9a9 9 0 019-9"></path>
							</svg>
							web hosting
						</h3>
						<ul class="space-y-2 nb-text-muted">
							<li>• custom subdomains</li>
							<li>• https certificates</li>
							<li>• port forwarding</li>
							<li>• reverse proxy setup</li>
						</ul>
					</div>
					
					<div class="nb-card text-left">
						<h3 class="nb-title text-xl mb-4 flex items-center gap-3">
							<svg class="w-6 h-6 text-[var(--nb-success)]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 7v10c0 2.21 3.582 4 8 4s8-1.79 8-4V7M4 7c0 2.21 3.582 4 8 4s8-1.79 8-4M4 7c0-2.21 3.582-4 8-4s8 1.79 8 4"></path>
							</svg>
							persistent storage
						</h3>
						<ul class="space-y-2 nb-text-muted">
							<li>• your files are safe</li>
							<li>• automatic backups</li>
							<li>• data export options</li>
							<li>• version control ready</li>
						</ul>
					</div>
					
					<div class="nb-card text-left">
						<h3 class="nb-title text-xl mb-4 flex items-center gap-3">
							<svg class="w-6 h-6 text-[var(--nb-info)]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19.428 15.428a2 2 0 00-1.022-.547l-2.387-.477a6 6 0 00-3.86.517l-.318.158a6 6 0 01-3.86.517L6.05 15.21a2 2 0 00-1.806.547M8 4h8l-1 1v5.172a2 2 0 00.586 1.414l5 5c1.26 1.26.367 3.414-1.415 3.414H4.828c-1.782 0-2.674-2.154-1.414-3.414l5-5A2 2 0 009 10.172V5L8 4z"></path>
							</svg>
							development tools
						</h3>
						<ul class="space-y-2 nb-text-muted">
							<li>• node.js, python, go, rust</li>
							<li>• docker & docker-compose</li>
							<li>• git, vim, tmux</li>
							<li>• package managers</li>
						</ul>
					</div>
				</div>
			</div>

			<div class="nb-card-lg bg-gradient-to-br from-[var(--nb-primary)] to-[var(--nb-secondary)] text-white mb-16">
				<div class="text-center">
					<p class="text-xl mb-6 opacity-90">
						den is built for the student community. 
						sign in with your github account to get access!
					</p>
					<div class="flex items-center justify-center gap-4 text-sm opacity-75">
						<div class="flex items-center gap-2">
							<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8c-1.657 0-3 .895-3 2s1.343 2 3 2 3 .895 3 2-1.343 2-3 2m0-8c1.11 0 2.08.402 2.599 1M12 8V7m0 1v8m0 0v1m0-1c-1.11 0-2.08-.402-2.599-1"></path>
							</svg>
							completely free
						</div>
						<div class="flex items-center gap-2">
							<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 10V3L4 14h7v7l9-11h-7z"></path>
							</svg>
							instant setup
						</div>
						<div class="flex items-center gap-2">
							<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4.318 6.318a4.5 4.5 0 000 6.364L12 20.364l7.682-7.682a4.5 4.5 0 00-6.364-6.364L12 7.636l-1.318-1.318a4.5 4.5 0 00-6.364 0z"></path>
							</svg>
							made with love
						</div>
					</div>
				</div>
			</div>
			<div class="text-center">
				<h2 class="nb-title text-3xl mb-6">ready to get started?</h2>
				<button 
					class="nb-button nb-button-xl nb-button-primary text-xl px-12 py-4 mb-4"
					on:click={login}
				>
					<svg class="w-6 h-6" viewBox="0 0 16 16" fill="currentColor">
						<path d="M8 0C3.58 0 0 3.58 0 8a8 8 0 0 0 5.47 7.59c.4.07.55-.17.55-.38 0-.19-.01-.82-.01-1.49-2.01.37-2.53-.49-2.69-.94-.09-.23-.48-.94-.82-1.13-.28-.15-.68-.52-.01-.53.63-.01 1.08.58 1.23.82.72 1.21 1.87.87 2.33.66.07-.52.28-.87.51-1.07-1.78-.2-3.64-.89-3.64-3.95 0-.87.31-1.59.82-2.15-.08-.2-.36-1.02.08-2.12 0 0 .67-.21 2.2.82.64-.18 1.32-.27 2-.27.68 0 1.36.09 2 .27 1.53-1.04 2.2-.82 2.2-.82.44 1.1.16 1.92.08 2.12.51.56.82 1.27.82 2.15 0 3.07-1.87 3.75-3.65 3.95.29.25.54.73.54 1.48 0 1.07-.01 1.93-.01 2.19 0 .21.15.46.55.38A8.01 8.01 0 0 0 16 8c0-4.42-3.58-8-8-8Z"/>
					</svg>
					sign in with github
				</button>
			</div>
		</div>
	</main>
</div>
