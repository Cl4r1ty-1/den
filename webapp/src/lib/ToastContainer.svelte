<script>
	import Toast from './Toast.svelte'
	
	let toasts = []
	let nextId = 0
	
	export function addToast(message, type = 'info', duration = 4000) {
		const id = nextId++
		const toast = { id, message, type, duration }
		toasts = [...toasts, toast]
		
		return id
	}
	
	function removeToast(id) {
		toasts = toasts.filter(t => t.id !== id)
	}
	
	if (typeof window !== 'undefined') {
		window.showToast = addToast
	}
</script>

<div class="nb-toast-container">
	{#each toasts as toast (toast.id)}
		<Toast
			message={toast.message}
			type={toast.type}
			duration={toast.duration}
			onRemove={() => removeToast(toast.id)}
		/>
	{/each}
</div>
