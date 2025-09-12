<script>
	import Header from '../lib/Header.svelte'
	export let user_count = 0
	export let node_count = 0
	export let container_count = 0
</script>

<Header user={{ IsAdmin: true }} />
<div class="nb-container p-6">
	<h1 class="nb-title text-3xl mb-4">admin</h1>
	<div class="grid md:grid-cols-3 gap-4">
		<div class="nb-card text-center">
			<div class="text-4xl font-extrabold">{user_count}</div>
			<div class="text-slate-600">users</div>
		</div>
		<div class="nb-card text-center">
			<div class="text-4xl font-extrabold">{node_count}</div>
			<div class="text-slate-600">nodes</div>
		</div>
		<div class="nb-card text-center">
			<div class="text-4xl font-extrabold">{container_count}</div>
			<div class="text-slate-600">containers</div>
		</div>
	</div>
</div>

