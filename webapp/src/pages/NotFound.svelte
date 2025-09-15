<script lang="ts">
	import Header from '../lib/Header.svelte'
	
	export let user: any = null
	
	const messages = [
		"oops! this page went on vacation",
		"404: page not found (but your vibes are immaculate)",
		"this page is hiding in another dimension",
		"error 404: page.exe has stopped working",
		"looks like this page took a wrong turn at albuquerque",
		"page not found - it's probably getting coffee",
		"404: this page is as lost as you are",
		"the page you're looking for is in another castle"
	]
	
	const randomMessage = messages[Math.floor(Math.random() * messages.length)]
</script>

<div class="min-h-screen bg-background text-foreground">
	<Header {user} currentPage="404" />
	
	<main class="max-w-4xl mx-auto p-6">
		<div class="text-center py-16">
			<div class="mb-8">
				<div class="text-9xl font-heading text-main border-8 border-border bg-secondary-background p-8 inline-block shadow-shadow transform -rotate-2">
					404
				</div>
			</div>
			
			<div class="mb-8">
				<h1 class="text-4xl font-heading mb-4 transform rotate-1">
					{randomMessage}
				</h1>
				<p class="text-xl text-foreground/70 max-w-2xl mx-auto">
					the page you're looking for doesn't exist, but hey, at least you found this delightfully broken page!
				</p>
			</div>
			
			<div class="grid md:grid-cols-3 gap-6 mb-12 max-w-3xl mx-auto">
				<a 
					href="/"
					class="bg-main text-main-foreground border-2 border-border p-6 text-center font-heading hover:translate-x-1 hover:translate-y-1 transition-transform shadow-shadow group"
				>
					<div class="w-12 h-12 mx-auto mb-3 bg-background border-2 border-border flex items-center justify-center group-hover:rotate-12 transition-transform">
						<svg class="w-6 h-6 text-foreground" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0a1 1 0 001-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 001 1m-6 0h6"></path>
						</svg>
					</div>
					<div class="text-lg">go home</div>
					<div class="text-sm text-main-foreground/70">back to safety</div>
				</a>
				
				{#if user}
					<a 
						href="/user/dashboard"
						class="bg-chart-2 text-main-foreground border-2 border-border p-6 text-center font-heading hover:translate-x-1 hover:translate-y-1 transition-transform shadow-shadow group"
					>
						<div class="w-12 h-12 mx-auto mb-3 bg-background border-2 border-border flex items-center justify-center group-hover:rotate-12 transition-transform">
							<svg class="w-6 h-6 text-foreground" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10"></path>
							</svg>
						</div>
						<div class="text-lg">dashboard</div>
						<div class="text-sm text-main-foreground/70">manage your stuff</div>
					</a>
				{:else}
					<a 
						href="/login"
						class="bg-chart-2 text-main-foreground border-2 border-border p-6 text-center font-heading hover:translate-x-1 hover:translate-y-1 transition-transform shadow-shadow group"
					>
						<div class="w-12 h-12 mx-auto mb-3 bg-background border-2 border-border flex items-center justify-center group-hover:rotate-12 transition-transform">
							<svg class="w-6 h-6 text-foreground" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 16l-4-4m0 0l4-4m-4 4h14m-5 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h7a3 3 0 013 3v1"></path>
							</svg>
						</div>
						<div class="text-lg">login</div>
						<div class="text-sm text-main-foreground/70">get started</div>
					</a>
				{/if}
				
				<button 
					on:click={() => history.back()}
					class="bg-chart-4 text-main-foreground border-2 border-border p-6 text-center font-heading hover:translate-x-1 hover:translate-y-1 transition-transform shadow-shadow group"
				>
					<div class="w-12 h-12 mx-auto mb-3 bg-background border-2 border-border flex items-center justify-center group-hover:rotate-12 transition-transform">
						<svg class="w-6 h-6 text-foreground" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 19l-7-7m0 0l7-7m-7 7h18"></path>
						</svg>
					</div>
					<div class="text-lg">go back</div>
					<div class="text-sm text-main-foreground/70">where you came from</div>
				</button>
			</div>
			
			<div class="bg-secondary-background border-2 border-border p-6 shadow-shadow max-w-2xl mx-auto">
				<pre class="font-mono text-sm text-foreground/70 whitespace-pre-wrap text-left">
    ¯\_(ツ)_/¯
    
    "i have no idea where that page went"
        - the server, probably
				</pre>
			</div>
			<div class="mt-12 text-sm text-foreground/50">
				<p>if you think this is a mistake, try refreshing the page or</p>
				<p>check the url for typos (we've all been there)</p>
			</div>
		</div>
	</main>
</div>
