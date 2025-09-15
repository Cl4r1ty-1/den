<script lang="ts">
	import Header from '../lib/Header.svelte'
	
	export let user: { display_name: string; username: string; email: string; approval_status: string; rejection_reason?: string }
</script>

<div class="min-h-screen bg-background text-foreground">
	<Header {user} currentPage="pending" />
	
	<main class="max-w-4xl mx-auto p-6">
		<div class="text-center py-16">
			{#if user.approval_status === 'pending'}
				<div class="mb-8">
					<div class="w-24 h-24 mx-auto mb-6 bg-chart-3 border-2 border-border flex items-center justify-center">
						<svg class="w-12 h-12 text-main-foreground" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"></path>
						</svg>
					</div>
					<h1 class="text-4xl font-heading mb-4">approval pending</h1>
					<p class="text-xl text-foreground/70 max-w-2xl mx-auto mb-8">
						hey {user.display_name}! your account is waiting for admin approval before you can create environments.
					</p>
				</div>
				
				<div class="bg-secondary-background border-2 border-border p-6 shadow-shadow max-w-2xl mx-auto mb-8">
					<div class="flex items-start gap-3">
						<svg class="w-5 h-5 mt-0.5 text-chart-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path>
						</svg>
						<div>
							<h3 class="font-heading font-bold mb-2">what happens next?</h3>
							<ul class="text-sm text-foreground/70 space-y-1 text-left">
								<li>• an admin will review your account request</li>
								<li>• you'll receive an email notification when approved</li>
								<li>• once approved, you can create development environments</li>
								<li>• approval usually takes 1-2 business days</li>
							</ul>
						</div>
					</div>
				</div>
				
				<div class="bg-background border-2 border-border p-4 max-w-md mx-auto">
					<h4 class="font-heading mb-2">account details</h4>
					<div class="text-sm text-foreground/70 space-y-1">
						<div><strong>username:</strong> {user.username}</div>
						<div><strong>email:</strong> {user.email}</div>
						<div><strong>status:</strong> <span class="text-chart-3">pending approval</span></div>
					</div>
				</div>
				
			{:else if user.approval_status === 'rejected'}
				<div class="mb-8">
					<div class="w-24 h-24 mx-auto mb-6 bg-chart-1 border-2 border-border flex items-center justify-center">
						<svg class="w-12 h-12 text-main-foreground" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"></path>
						</svg>
					</div>
					<h1 class="text-4xl font-heading mb-4">application rejected</h1>
					<p class="text-xl text-foreground/70 max-w-2xl mx-auto mb-8">
						unfortunately, your account application was not approved.
					</p>
				</div>
				
				{#if user.rejection_reason}
					<div class="bg-secondary-background border-2 border-border p-6 shadow-shadow max-w-2xl mx-auto mb-8">
						<h3 class="font-heading font-bold mb-2">reason for rejection</h3>
						<p class="text-foreground/70">{user.rejection_reason}</p>
					</div>
				{/if}
				
				<div class="bg-chart-2 text-main-foreground border-2 border-border p-6 shadow-shadow max-w-2xl mx-auto">
					<div class="flex items-start gap-3">
						<svg class="w-5 h-5 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path>
						</svg>
						<div>
							<h4 class="font-heading font-bold mb-2">what can you do?</h4>
							<ul class="text-sm opacity-90 space-y-1 text-left">
								<li>• contact an administrator to discuss your application</li>
								<li>• address any concerns mentioned in the rejection reason</li>
								<li>• you may be able to reapply in the future</li>
							</ul>
						</div>
					</div>
				</div>
			{/if}
		</div>
	</main>
</div>
