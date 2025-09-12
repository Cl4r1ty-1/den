<script>
	export let user
	export let container
	export let subdomains = []
</script>

<div class="p-6 max-w-5xl mx-auto">
	<h1 class="text-2xl font-bold mb-4">welcome, {user?.DisplayName}</h1>
	<div class="grid md:grid-cols-2 gap-4">
		<div class="nb-card">
			<h2 class="font-semibold mb-2">status</h2>
			{#if container}
				<div>running</div>
				<div class="mt-2 text-sm text-gray-700">container id: {container.ID}</div>
			{:else}
				<div>no environment</div>
			{/if}
		</div>
		<div class="nb-card">
			<h2 class="font-semibold mb-2">subdomains</h2>
			{#if subdomains.length}
				<ul>
					{#each subdomains as s}
						<li>{s.Subdomain} → {s.TargetPort}</li>
					{/each}
				</ul>
			{:else}
				<div>no subdomains</div>
			{/if}
		</div>
	</div>
</div>

