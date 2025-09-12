<script>
	import Header from '../lib/Header.svelte'
	export let user
	export let container
	export let subdomains = []

	async function createContainer() {
		await fetch('/user/container/create', { method: 'POST', headers: { 'Content-Type': 'application/json' } })
		location.reload()
	}
	async function getNewPort() {
		await fetch('/user/container/ports/new', { method: 'POST', headers: { 'Content-Type': 'application/json' } })
		location.reload()
	}
</script>

<Header {user} />
<div class="nb-container p-6">
	<h1 class="nb-title text-3xl mb-4">welcome, {user?.DisplayName}</h1>
	<div class="grid md:grid-cols-2 gap-4">
		<div class="nb-card">
			<h2 class="font-semibold mb-2">status</h2>
			{#if container}
				<div class="nb-pill">running</div>
				<div class="mt-3 text-sm text-slate-700 space-y-1">
					<div><span class="font-semibold">container id:</span> {container.ID}</div>
					<div><span class="font-semibold">ip:</span> {container.IPAddress}</div>
					<div><span class="font-semibold">memory:</span> {container.MemoryMB}MB</div>
					<div><span class="font-semibold">cpu:</span> {container.CPUCores}</div>
					<div><span class="font-semibold">storage:</span> {container.StorageGB}GB</div>
				</div>
				<div class="mt-3 flex gap-2">
					<button class="nb-button" on:click={getNewPort}>get new port</button>
					<a class="nb-button" href="/user/ssh-setup">configure ssh</a>
				</div>
			{:else}
				<div class="nb-pill">no environment</div>
				<p class="text-sm mt-2">create an environment to get started.</p>
				<div class="mt-3">
					<button class="nb-button" on:click={createContainer}>create container</button>
				</div>
			{/if}
		</div>
		<div class="nb-card">
			<h2 class="font-semibold mb-2">subdomains</h2>
			{#if subdomains.length}
				<ul class="space-y-2">
					{#each subdomains as s}
						<li class="flex justify-between items-center bg-[var(--nb-muted)] rounded-md p-2 border-2 border-[var(--nb-accent)]">
							<div class="font-mono text-sm">{s.Subdomain} → {s.TargetPort}</div>
						</li>
					{/each}
				</ul>
			{:else}
				<div class="text-sm text-slate-600">no subdomains yet.</div>
			{/if}
		</div>
	</div>
</div>

