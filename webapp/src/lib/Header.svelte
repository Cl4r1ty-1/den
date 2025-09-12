<script>
	export let user = null
</script>

<header class="bg-white border-b-2 border-[var(--nb-accent)] shadow-[var(--nb-shadow-sm)]">
	<div class="nb-container flex items-center justify-between py-3">
		<a href="/user/dashboard" class="nb-title text-xl">den</a>
		<nav class="flex items-center gap-4 text-sm">
			<a href="/user/dashboard" class="nb-button px-3 py-1">dashboard</a>
			<a href="/user/subdomains" class="nb-button px-3 py-1">subdomains</a>
			{#if user?.IsAdmin}
				<a href="/admin" class="nb-button px-3 py-1">admin</a>
			{/if}
		</nav>
	</div>
</header>
