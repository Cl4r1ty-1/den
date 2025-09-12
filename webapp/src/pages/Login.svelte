<script>
	function login() { window.location.href = '/auth/github' }
</script>

<div class="min-h-screen flex items-center justify-center p-6 relative overflow-hidden">
	<div class="absolute inset-0 opacity-5">
		<div class="absolute inset-0" style="background-image: radial-gradient(circle at 25px 25px, var(--nb-accent) 2px, transparent 0), radial-gradient(circle at 75px 75px, var(--nb-accent) 2px, transparent 0); background-size: 100px 100px;"></div>
	</div>
	
	<div class="nb-card-lg max-w-lg w-full text-center relative z-10">
		<div class="mb-8">
			<h1 class="nb-title text-6xl mb-3">
				<span class="text-[var(--nb-primary)]">d</span><span class="text-[var(--nb-secondary)]">e</span><span class="text-[var(--nb-info)]">n</span>
			</h1>
			<p class="nb-subtitle text-xl mb-2">your cozy *nix home</p>
			<p class="nb-text-muted">get your own container with ssh access</p>
		</div>
		
		<div class="grid grid-cols-3 gap-4 mb-8">
			<div class="nb-card text-center">
				<div class="w-12 h-12 mx-auto mb-2 bg-[var(--nb-success)] rounded-lg flex items-center justify-center">
					<svg class="w-6 h-6 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z"></path>
					</svg>
				</div>
				<h3 class="font-bold text-sm">secure</h3>
				<p class="text-xs nb-text-muted">isolated environment</p>
			</div>
			
			<div class="nb-card text-center">
				<div class="w-12 h-12 mx-auto mb-2 bg-[var(--nb-info)] rounded-lg flex items-center justify-center">
					<svg class="w-6 h-6 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 10V3L4 14h7v7l9-11h-7z"></path>
					</svg>
				</div>
				<h3 class="font-bold text-sm">fast</h3>
				<p class="text-xs nb-text-muted">instant deployment</p>
			</div>
			
			<div class="nb-card text-center">
				<div class="w-12 h-12 mx-auto mb-2 bg-[var(--nb-primary)] rounded-lg flex items-center justify-center">
					<svg class="w-6 h-6 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"></path>
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"></path>
					</svg>
				</div>
				<h3 class="font-bold text-sm">powerful</h3>
				<p class="text-xs nb-text-muted">full linux access</p>
			</div>
		</div>
		
		<button 
			class="nb-button nb-button-lg nb-button-primary w-full mb-6 text-lg" 
			on:click={login}
		>
			<svg class="w-6 h-6" viewBox="0 0 16 16" fill="currentColor">
				<path d="M8 0C3.58 0 0 3.58 0 8a8 8 0 0 0 5.47 7.59c.4.07.55-.17.55-.38 0-.19-.01-.82-.01-1.49-2.01.37-2.53-.49-2.69-.94-.09-.23-.48-.94-.82-1.13-.28-.15-.68-.52-.01-.53.63-.01 1.08.58 1.23.82.72 1.21 1.87.87 2.33.66.07-.52.28-.87.51-1.07-1.78-.2-3.64-.89-3.64-3.95 0-.87.31-1.59.82-2.15-.08-.2-.36-1.02.08-2.12 0 0 .67-.21 2.2.82.64-.18 1.32-.27 2-.27.68 0 1.36.09 2 .27 1.53-1.04 2.2-.82 2.2-.82.44 1.1.16 1.92.08 2.12.51.56.82 1.27.82 2.15 0 3.07-1.87 3.75-3.65 3.95.29.25.54.73.54 1.48 0 1.07-.01 1.93-.01 2.19 0 .21.15.46.55.38A8.01 8.01 0 0 0 16 8c0-4.42-3.58-8-8-8Z"/>
			</svg>
			continue with GitHub
		</button>
		
		<div class="nb-card bg-[var(--nb-surface-alt)]">
			<p class="text-sm nb-text-muted">
				<strong>new to den?</strong> sign in with your GitHub account to get started. 
				you'll get access to a cozy unix environment with all the tools you need.
			</p>
		</div>
	</div>
</div>

