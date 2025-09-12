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
		class="nb-modal-backdrop" 
		on:click={handleBackdropClick}
		on:keydown={handleKeydown}
		role="dialog"
		aria-modal="true"
		tabindex="-1"
	>
		<div class="nb-modal {sizes[size]} w-full">
			{#if title}
				<div class="nb-modal-header">
					<div class="flex items-center justify-between">
						<h3 class="nb-title text-xl">{title}</h3>
						<button 
							class="nb-button nb-button-sm" 
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
			
			<div class="nb-modal-body">
				<slot />
			</div>
			
			{#if $$slots.footer}
				<div class="nb-modal-footer">
					<slot name="footer" />
				</div>
			{/if}
		</div>
	</div>
{/if}
