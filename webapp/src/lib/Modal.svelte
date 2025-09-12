<script>
	export let show = false
	export let title = ''
	export let size = 'md'
	export let onClose = () => {}
	
	const sizes = {
		sm: 'max-w-md',
		md: 'max-w-lg',
		lg: 'max-w-2xl',
		xl: 'max-w-4xl'
	}
	
	function handleBackdropClick(e) {
		if (e.target === e.currentTarget) {
			onClose()
		}
	}
	
	function handleKeydown(e) {
		if (e.key === 'Escape') {
			onClose()
		}
	}
</script>

{#if show}
	<div 
		class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-overlay" 
		on:click={handleBackdropClick}
		on:keydown={handleKeydown}
		role="dialog"
		aria-modal="true"
		tabindex="-1"
	>
		<div class="bg-secondary-background border-2 border-border shadow-shadow {sizes[size]} w-full">
			{#if title}
				<div class="p-6 border-b-2 border-border">
					<div class="flex items-center justify-between">
						<h3 class="text-xl font-heading">{title}</h3>
						<button 
							class="bg-foreground/10 border-2 border-border p-2 hover:translate-x-1 hover:translate-y-1 transition-transform" 
							on:click={onClose}
							aria-label="Close modal"
						>
							<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"></path>
							</svg>
						</button>
					</div>
				</div>
			{/if}
			
			<div class="p-6">
				<slot />
			</div>
			
			{#if $$slots.footer}
				<div class="p-6 border-t-2 border-border">
					<slot name="footer" />
				</div>
			{/if}
		</div>
	</div>
{/if}
