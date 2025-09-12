<script>
	export let user_count = 0
	export let node_count = 0
	export let container_count = 0
</script>

<div class="p-6 max-w-6xl mx-auto">
	<h1 class="text-2xl font-bold mb-4">admin</h1>
	<div class="grid md:grid-cols-3 gap-4">
		<div class="nb-card text-center"><div class="text-3xl font-extrabold">{user_count}</div><div>users</div></div>
		<div class="nb-card text-center"><div class="text-3xl font-extrabold">{node_count}</div><div>nodes</div></div>
		<div class="nb-card text-center"><div class="text-3xl font-extrabold">{container_count}</div><div>containers</div></div>
	</div>
</div>

