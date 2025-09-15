<script>
	import Modal from '../lib/Modal.svelte'
	import ToastContainer from '../lib/ToastContainer.svelte'
	
	export let user
	export let quiz_questions = []
	
	let acceptTOS = false
	let acceptPrivacy = false
	let showQuizModal = false
	let quizAnswers = {}
	let toastContainer
	
	function startQuiz() {
		if (!acceptTOS || !acceptPrivacy) {
			toastContainer.addToast('Please accept both policies to continue', 'danger')
			return
		}
		
		if (!quiz_questions || quiz_questions.length === 0) {
			toastContainer.addToast('No quiz questions available. Please reload the page.', 'danger')
			return
		}
		
		quizAnswers = {}
		quiz_questions.forEach(q => {
			quizAnswers[q.id] = ''
		})
		
		showQuizModal = true
	}
	
	async function submitQuiz() {
		const answers = Object.values(quizAnswers)
		if (answers.some(answer => !answer.trim())) {
			toastContainer.addToast('Please answer all questions', 'danger')
			return
		}
		
		const answersArray = Object.entries(quizAnswers).map(([id, answer]) => ({
			id: parseInt(id),
			answer: answer.trim()
		}))
		
		const res = await fetch('/user/aup/accept', {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({
				accept_tos: true,
				accept_privacy: true,
				answers: answersArray
			})
		})
		
		const data = await res.json()
		if (data.error) {
			toastContainer.addToast(data.error, 'danger')
			return
		}
		
		showQuizModal = false
		window.location.href = '/user/dashboard'
	}
</script>

<div class="min-h-screen bg-background text-foreground py-8">
	<div class="max-w-4xl mx-auto px-6">
		<div class="text-center mb-8">
			<h1 class="text-4xl font-heading mb-3">
				welcome, <span class="text-main">{user.display_name}</span>!
			</h1>
			<p class="text-xl text-foreground/70">please review and accept our policies to continue</p>
		</div>
		<div class="grid md:grid-cols-2 gap-6 mb-8">
			<div class="bg-secondary-background border-2 border-border p-6 shadow-shadow">
				<div class="flex items-center gap-3 mb-4">
					<div class="w-12 h-12 bg-chart-1 border-2 border-border flex items-center justify-center">
						<svg class="w-6 h-6 text-main-foreground" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z"></path>
						</svg>
					</div>
					<h2 class="text-xl font-heading">acceptable use policy</h2>
				</div>
				
				<div class="bg-background border-2 border-border p-4 mb-4 text-sm text-foreground/70">
					<p class="mb-3">our AUP ensures a safe and fair environment for everyone. key points:</p>
					<ul class="space-y-1">
						<li>• no cryptocurrency mining</li>
						<li>• no malicious activities</li>
						<li>• respect resource limits</li>
						<li>• no illegal content</li>
						<li>• be respectful to others</li>
					</ul>
				</div>
				
				<details class="mb-4">
					<summary class="cursor-pointer font-heading mb-2">read full policy</summary>
					<div class="bg-background border-2 border-border p-4 text-xs text-foreground/70 max-h-60 overflow-y-auto">
						<h3 class="font-bold mb-2">Acceptable Use Policy (AUP)</h3>
						<p class="mb-2"><strong>Effective Date:</strong> 2025-08-10</p>
						
						<h4 class="font-bold mt-3 mb-1">1. Introduction</h4>
						<p class="mb-2">This Acceptable Use Policy governs your use of the "den" hosting service. By using the Service, you agree to abide by this Policy.</p>
						
						<h4 class="font-bold mt-3 mb-1">2. Prohibited Activities</h4>
						<p class="mb-1"><strong>Cryptocurrency Mining:</strong> Any form of cryptocurrency mining is strictly prohibited.</p>
						<p class="mb-1"><strong>Malicious Activities:</strong> No malware, viruses, or harmful code distribution.</p>
						<p class="mb-1"><strong>Resource Abuse:</strong> Excessive CPU, memory, or network usage that impacts other users.</p>
						<p class="mb-1"><strong>Illegal Content:</strong> No illegal, defamatory, or harmful content.</p>
						
						<h4 class="font-bold mt-3 mb-1">3. Enforcement</h4>
						<p>Violations may result in account suspension or termination. We reserve the right to investigate suspected violations.</p>
					</div>
				</details>
				
				<label class="flex items-center gap-2 cursor-pointer">
					<input id="accept_tos" type="checkbox" bind:checked={acceptTOS} class="w-4 h-4">
					<label for="accept_tos" class="text-sm">I have read and agree to the Acceptable Use Policy</label>
				</label>
			</div>
			
			<div class="bg-secondary-background border-2 border-border p-6 shadow-shadow">
				<div class="flex items-center gap-3 mb-4">
					<div class="w-12 h-12 bg-chart-2 border-2 border-border flex items-center justify-center">
						<svg class="w-6 h-6 text-main-foreground" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z"></path>
						</svg>
					</div>
					<h2 class="text-xl font-heading">privacy policy</h2>
				</div>
				
				<div class="bg-background border-2 border-border p-4 mb-4 text-sm text-foreground/70">
					<p class="mb-3">we respect your privacy and protect your data. what we collect:</p>
					<ul class="space-y-1">
						<li>• account information (GitHub profile)</li>
						<li>• container metadata</li>
						<li>• access logs for security</li>
						<li>• usage statistics</li>
					</ul>
					<p class="mt-3">we never sell your data or use it for advertising.</p>
				</div>
				
				<details class="mb-4">
					<summary class="cursor-pointer font-heading mb-2">read full policy</summary>
					<div class="bg-background border-2 border-border p-4 text-xs text-foreground/70 max-h-60 overflow-y-auto">
						<h3 class="font-bold mb-2">Privacy Policy</h3>
						<p class="mb-2"><strong>Effective Date:</strong> 2025-08-10</p>
						
						<h4 class="font-bold mt-3 mb-1">Information We Collect</h4>
						<p class="mb-1"><strong>Account Information:</strong> GitHub ID, username, display name, email</p>
						<p class="mb-1"><strong>Service Metadata:</strong> Container ID, ports, subdomains, configuration</p>
						<p class="mb-1"><strong>Logs:</strong> Access logs, security logs, system metrics</p>
						
						<h4 class="font-bold mt-3 mb-1">How We Use Information</h4>
						<p class="mb-1">• To provide and maintain the service</p>
						<p class="mb-1">• To secure and protect the platform</p>
						<p class="mb-1">• To communicate important updates</p>
						
						<h4 class="font-bold mt-3 mb-1">Data Retention</h4>
						<p class="mb-1">Account data is retained while your account is active. Logs are retained for security purposes (30-90 days).</p>
						
						<h4 class="font-bold mt-3 mb-1">Your Rights</h4>
						<p>You can request access, correction, or deletion of your data. Contact us at support@hack.ngo</p>
					</div>
				</details>
				
				<div class="flex items-center gap-2">
					<input id="accept_privacy" type="checkbox" bind:checked={acceptPrivacy} class="w-4 h-4">
					<label for="accept_privacy" class="text-sm">I have read and agree to the Privacy Policy</label>
				</div>
			</div>
		</div>

		<div class="text-center">
			<button 
				class="bg-main text-main-foreground border-2 border-border px-8 py-3 text-lg font-heading shadow-shadow hover:translate-x-1 hover:translate-y-1 transition-transform disabled:opacity-50 disabled:cursor-not-allowed disabled:transform-none"
				on:click={startQuiz}
				disabled={!acceptTOS || !acceptPrivacy}
			>
				<svg class="w-5 h-5 inline mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"></path>
				</svg>
				continue to verification quiz
			</button>
		</div>

		<div class="bg-secondary-background border-2 border-border p-6 shadow-shadow mt-8">
			<div class="flex items-start gap-3">
				<div class="w-12 h-12 bg-chart-3 border-2 border-border flex items-center justify-center">
					<svg class="w-6 h-6 text-main-foreground" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path>
					</svg>
				</div>
				<div>
					<h4 class="text-xl font-heading mb-2">why do we need this?</h4>
					<p class="text-foreground/70">
						by accepting these policies, you help us maintain a safe, secure, and fair environment for all users. 
						the verification quiz ensures you've read and understood our guidelines.
					</p>
				</div>
			</div>
		</div>
	</div>
</div>

<Modal 
	show={showQuizModal} 
	title="verification quiz" 
	size="lg"
	onClose={() => showQuizModal = false}
>
	<div class="mb-4">
		<div class="bg-chart-3 text-main-foreground border-2 border-border p-4 shadow-shadow">
			<div class="flex items-center gap-2">
				<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9.663 17h4.673M12 3v1m6.364 1.636l-.707.707M21 12h-1M4 12H3m3.343-5.657l-.707-.707m2.828 9.9a5 5 0 117.072 0l-.548.547A3.374 3.374 0 0014 18.469V19a2 2 0 11-4 0v-.531c0-.895-.356-1.754-.988-2.386l-.548-.547z"></path>
				</svg>
				<span class="font-heading">prove you read the policies!</span>
			</div>
			<p class="text-sm mt-1">answer these questions to show you understand our guidelines</p>
		</div>
	</div>
	
	<form on:submit|preventDefault={submitQuiz} class="space-y-6">
		{#each quiz_questions as question, index}
			<div>
				<label class="block text-sm font-heading mb-2" for={`quiz_${question.id}`}>
					{index + 1}. {question.prompt}
				</label>
				<input 
					id={`quiz_${question.id}`}
					type="text" 
					bind:value={quizAnswers[question.id]}
					placeholder="your answer"
					class="w-full bg-background border-2 border-border p-3"
					required
				>
			</div>
		{/each}
	</form>
	
	<div slot="footer" class="flex gap-3">
		<button class="bg-foreground/10 border-2 border-border px-4 py-2 font-heading hover:translate-x-1 hover:translate-y-1 transition-transform" on:click={() => showQuizModal = false}>cancel</button>
		<button class="bg-main text-main-foreground border-2 border-border px-4 py-2 font-heading shadow-shadow hover:translate-x-1 hover:translate-y-1 transition-transform" on:click={submitQuiz}>submit answers</button>
	</div>
</Modal>

<ToastContainer bind:this={toastContainer} />
