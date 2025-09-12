<script>
	import { router } from '@inertiajs/svelte'
	function login() { window.location.href = '/auth/github' }
</script>

<div class="min-h-screen flex items-center justify-center p-6">
	<div class="nb-card max-w-md w-full text-center">
		<h1 class="text-3xl font-extrabold mb-2">den</h1>
		<p class="text-gray-700 mb-6">den - a cozy *nix system</p>
		<button class="nb-button w-full" on:click={login}>continue with GitHub</button>
	</div>
</div>

