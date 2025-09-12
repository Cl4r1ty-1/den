<script>
	export let user = null
	export let currentPage = 'dashboard'
</script>

<header class="bg-white border-b-3 border-[var(--nb-accent)] nb-shadow-sm sticky top-0 z-40">
	<div class="nb-container flex items-center justify-between py-4">
		<a href="/user/dashboard" class="nb-title text-2xl hover:scale-105 transition-transform duration-150">
			<span class="text-[var(--nb-primary)]">den</span>
		</a>
		
		<nav class="flex items-center gap-3">
			<a 
				href="/user/dashboard" 
				class="nb-button nb-button-sm {currentPage === 'dashboard' ? 'nb-button-primary' : ''}"
			>
				<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2H5a2 2 0 00-2-2z"></path>
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 5a2 2 0 012-2h4a2 2 0 012 2v6H8V5z"></path>
				</svg>
				dashboard
			</a>
			
			<a 
				href="/user/subdomains" 
				class="nb-button nb-button-sm {currentPage === 'subdomains' ? 'nb-button-primary' : ''}"
			>
				<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 12a9 9 0 01-9 9m9-9a9 9 0 00-9-9m9 9H3m9 9a9 9 0 01-9-9m9 9c1.657 0 3-4.03 3-9s-1.343-9-3-9m0 18c-1.657 0-3-4.03-3-9s1.343-9 3-9m-9 9a9 9 0 019-9"></path>
				</svg>
				subdomains
			</a>
			
			{#if user?.IsAdmin}
				<a 
					href="/admin" 
					class="nb-button nb-button-sm nb-button-warning {currentPage === 'admin' ? 'nb-button-primary' : ''}"
				>
					<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"></path>
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"></path>
					</svg>
					admin
				</a>
			{/if}
			
			{#if user}
				<div class="flex items-center gap-2 ml-2 pl-2 border-l-2 border-[var(--nb-border)]">
					<div class="text-sm">
						<div class="font-bold nb-text">{user.DisplayName}</div>
						<div class="nb-text-muted text-xs">@{user.Username}</div>
					</div>
					<a href="/logout" class="nb-button nb-button-sm nb-button-danger">
						<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 16l4-4m0 0l-4-4m4 4H7m6 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h4a3 3 0 013 3v1"></path>
						</svg>
					</a>
				</div>
			{/if}
		</nav>
	</div>
</header>
