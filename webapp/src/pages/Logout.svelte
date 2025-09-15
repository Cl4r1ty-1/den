<script>
	import { onMount } from 'svelte'
	let seconds = 5
	onMount(() => {
		const timer = setInterval(() => {
			seconds -= 1
			if (seconds <= 0) {
				clearInterval(timer)
				window.location.href = '/'
			}
		}, 1000)
	})
</script>

<div class="min-h-screen bg-background text-foreground flex items-center justify-center p-6">
	<div class="bg-secondary-background border-2 border-border p-8 max-w-md w-full text-center shadow-shadow">
		<h1 class="text-3xl font-heading mb-2">you've been logged out</h1>
		<p class="text-foreground/70 mb-6">returning to home page in {seconds} seconds…</p>
		<a href="/" class="bg-main text-main-foreground border-2 border-border px-4 py-2 font-heading inline-block hover:translate-x-1 hover:translate-y-1 transition-transform shadow-shadow">
			go now
		</a>
	</div>
</div>
