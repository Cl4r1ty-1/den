<script>
	import { onMount } from 'svelte'
	
	export let message = ''
	export let type = 'info'
	export let duration = 4000
	export let onRemove = () => {}
	
	onMount(() => {
		if (duration > 0) {
			const timer = setTimeout(() => {
				onRemove()
			}, duration)
			
			return () => clearTimeout(timer)
		}
	})
</script>

<div class="nb-toast nb-toast-{type}">
	<div class="flex items-center justify-between gap-3">
		<div class="flex items-center gap-2">
			{#if type === 'success'}
				<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"></path>
				</svg>
			{:else if type === 'danger'}
				<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path>
				</svg>
			{:else if type === 'warning'}
				<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.964-.833-2.732 0L3.732 16.5c-.77.833.192 2.5 1.732 2.5z"></path>
				</svg>
			{:else}
				<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path>
				</svg>
			{/if}
			
			<span class="font-medium">{message}</span>
		</div>
		
		<button 
			class="opacity-70 hover:opacity-100 transition-opacity"
			on:click={onRemove}
			aria-label="Dismiss"
		>
			<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"></path>
			</svg>
		</button>
	</div>
</div>
