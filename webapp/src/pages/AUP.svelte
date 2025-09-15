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
						<pre class="whitespace-pre-wrap font-sans">
							<h3>Acceptable Use Policy (AUP)</h3>

**Effective Date:** 2025-08-10

### 1. Introduction

This Acceptable Use Policy (the "Policy" or "AUP") governs your use of the "den" hosting service (the "Service"). Its purpose is to protect the Service, our users, and the wider internet community from irresponsible, abusive, or illegal activities.

By using the Service, you agree to abide by this Policy. It is your responsibility to read and understand it. This AUP is a core part of your agreement with us. Failure to comply constitutes a material breach of our terms and may result in the actions outlined in the "Enforcement" section of this policy.

Common sense and respect for others are the best guides for what is acceptable.

### 2. General Principles & Appropriate Use

The Service is designed for individuals to learn, experiment, and build personal, non-commercial projects. Appropriate uses include:

*   Developing and hosting lightweight hobby applications, personal websites, and bots.
*   Learning about Linux, networking, and software development in a hands-on environment.
*   Participating in a community of developers and respecting the shared nature of the infrastructure.

### 3. Prohibited Activities

The following activities are strictly prohibited. This list is not exhaustive, but serves as a guideline for unacceptable conduct.

**a. Illegal and Harmful Content:**
You may not use the Service to create, store, transmit, or display any content that violates the laws of the United Kingdom or your local jurisdiction. This includes, but is not limited to:
*   Content that is defamatory, obscene, harassing, or threatening.
*   Material protected by copyright, trademark, or other intellectual property rights used without proper authorization.
*   Content that facilitates or promotes illegal activities, such as fraud, drug dealing, or trafficking.
*   Hate speech or content that promotes violence or discrimination against any individual or group.

**b. System and Network Abuse:**
You are responsible for ensuring your use of the Service does not harm the platform or other users. Prohibited actions include:
*   **Cryptocurrency Mining:** Engaging in any form of cryptocurrency mining, cryptojacking, or participating in mining pools.
*   **Malicious Activities:** Distributing, hosting, or executing malware, viruses, worms, Trojan horses, or any other code designed to disrupt, damage, or gain unauthorized access to any system.
*   **Denial of Service (DoS):** Launching or participating in any form of DoS attack, network flooding, or other activity designed to interfere with the service of any user, host, or network.
*   **Spam and Unsolicited Communication:** Sending or assisting in the transmission of unsolicited bulk email (spam), "mail-bombing," or other harassing communications.

**c. Security Violations:**
You may not take any action to compromise the security of the Service or any other user's environment. This includes:
*   **Unauthorized Access:** Attempting to access, probe, scan, or test the vulnerability of any account, system, or network without explicit permission. This includes using tools designed for compromising security, such as password crackers or network scanning tools.
*   **Bypassing Limitations:** Attempting to tamper with or bypass any security measures, monitoring, or resource limits put in place by the Service.
*   **Falsification of Origin:** Forging TCP/IP packet headers, email headers, or any part of a message to disguise its origin or route.

**d. Resource Abuse:**
The Service operates on shared infrastructure. You must not consume a disproportionate amount of system resources in a way that negatively impacts other users.
*   **Resource Hoarding:** Sustained and excessive use of CPU, RAM, disk I/O, or network bandwidth is prohibited. This is not about short bursts of activity but about continuous, high-load processes that degrade server performance for others.
*   **Unattended Processes:** Running stand-alone, unattended server-side processes, daemons, or bots that consume significant resources is not permitted without prior arrangement.
*   **File Sharing:** Running services for peer-to-peer (P2P) file sharing, such as BitTorrent trackers or clients, is forbidden.

### 4. User Responsibility

*   **Account Security:** You are solely responsible for maintaining the security of your account credentials. You must take reasonable precautions to prevent unauthorized access. Any activity originating from your account will be considered your responsibility.
*   **Provider Policies:** Your use of the Service must also comply with the acceptable use policies of our upstream infrastructure providers (e.g., Google Cloud).
*   **Compliance with Law:** You are responsible for ensuring that all content and activities within your container comply with all applicable laws and regulations.

### 5. Enforcement and Violations

We reserve the right to investigate any suspected violation of this Policy. When a breach occurs, we may take any action we deem appropriate, based on the severity of the violation. Actions may include:

*   **Issuing a formal warning** and requesting that the violation be corrected.
*   **Throttling or rate-limiting** the offending service or resource.
*   **Immediate temporary or permanent suspension** of your account and services, with or without prior notice.
*   **Removal of offending content** from our servers.
*   **Reporting illegal activities** to the relevant law enforcement authorities and providing them with necessary information.

We will be the sole arbiters of what constitutes a violation of this Policy. Our failure to enforce this policy in any given instance shall not be construed as a waiver of our right to do so in the future.

### 6. Reporting Violations

If you become aware of any violation of this AUP, whether by another user or an external party, please notify us immediately. To report a violation, please contact us at **abuse@hack.ngo**.

Please provide as much detail as possible, including any relevant logs, timestamps, and IP addresses, to assist in our investigation.

### 7. Policy Changes

We may revise this Acceptable Use Policy at any time by posting the updated version on our website. You are expected to check this page periodically to take notice of any changes. Your continued use of the Service after a change constitutes your acceptance of the new Policy.</pre>
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
						<pre class="whitespace-pre-wrap font-sans"># Privacy Policy

**Effective date:** 2025-08-10

This Privacy Policy explains how "den" (the "Service") collects, uses, and shares information about you when you use the Service.

*   **Controller**: The Service is operated by an individual sole trader based in the United Kingdom (UK). For the purposes of the UK General Data Protection Regulation (UK GDPR), the operator of the Service is the data controller. UK law (including the Computer Misuse Act 1990) applies.
*   **Infrastructure**: The Service is hosted in Google Cloud data centres. Some processing may occur outside the UK/European Economic Area (EEA), subject to appropriate safeguards.

## Information we collect

We collect only what we need to operate and secure the Service:

*   **Account information**: GitHub ID, username, display name, and email address provided during authentication.
*   **Service metadata**: Your unique container ID, assigned node/hostname, allocated ports, subdomains, and any configuration you set in the dashboard.
*   **Logs and diagnostics**: Access logs (IP address, user-agent, timestamps, request metadata), security/abuse logs, and system metrics necessary for operating, troubleshooting, and securing the platform.
*   **Cookies**: A single session cookie used exclusively to keep you signed in. We do not use tracking or advertising cookies.

## How we use your information

*   **To provide and maintain the Service**: This includes user isolation, port allocation, subdomain management, and reverse proxying.
*   **To secure and protect the Service**: We monitor for abuse, intrusion, or prohibited activities (e.g., crypto mining) and enforce system limits and policies.
*   **To communicate with you**: We use your contact information to send notices about your account, incidents, and important service changes.
*   **To comply with legal obligations**: We may process your data to comply with applicable laws or lawful requests from authorities.

## Legal bases for processing (UK GDPR)

We rely on the following legal bases to process your personal data:

*   **Contract**: To provide you with the Service and its features that you request, as outlined in our terms of service.
*   **Legitimate interests**: To operate, secure, and improve the Service; to prevent abuse; and to protect our users and infrastructure. Where we rely on legitimate interests, we have balanced these against your rights and freedoms.
*   **Legal obligation**: Where processing is necessary for us to comply with a legal requirement or a binding lawful request.
*   **Consent**: Where we explicitly ask for and you provide consent for a specific purpose (e.g., optional features). You can withdraw your consent at any time for future processing.

## Data retention

We keep personal data only as long as necessary for the purposes for which it was collected:

*   **Account and configuration data**: Retained for as long as your account is active.
*   **Logs and diagnostics**: Retained for a limited period appropriate for operations and security (e.g., rolling windows of 30-90 days). Security or incident logs may be retained for longer if required for an active investigation or by law.

## Deletion and data export

When you delete your account, you will have a 14-day grace period to download an export of your data. After this period, your account and associated personal data will be permanently deleted from our active systems. Data may persist for a limited time in encrypted backups until they are rotated, in accordance with our backup policy.

## Sharing and disclosures

We do not sell your personal data. We only share information under the following limited circumstances:

*   **Service providers ("processors")**: We use third-party providers for infrastructure and services necessary to run the platform (e.g., Google Cloud for hosting, Cloudflare for DNS). These providers are bound by strict contractual and security obligations.
*   **Law enforcement or authorities**: We may disclose information if we believe in good faith that it is reasonably necessary to comply with a law or lawful request, to protect the safety of any person, to prevent fraud or abuse, or to protect the Service's rights and property.

## International transfers

As we use global service providers like Google Cloud, your information may be processed in countries outside the UK/EEA. Where this occurs, we rely on appropriate legal safeguards, such as the provider's binding corporate rules, standard contractual clauses, and robust security certifications, to ensure your data is protected.

## Security

We use technical and organisational measures to protect your data, including isolation via LXC containers, network access controls, and continuous monitoring. However, no system can be perfectly secure. You are responsible for securing your own credentials and the content within your environment.

## Acceptable use and enforcement

Use of the Service is subject to our Acceptable Use Policy (AUP). Prohibited activities include, without limitation: crypto mining, malware distribution, denial-of-service attacks, spamming, and any attempts to bypass system security or limits. We reserve the right to throttle, suspend, or terminate accounts for suspected fraud, abuse, or illegal activity, and to cooperate with lawful investigations.

## Your rights (UK GDPR)

You have rights over your personal data. Depending on your location, these may include the right to:

*   **Access** the personal data we hold about you.
*   Request **correction** of inaccurate data or **deletion** of your data.
*   Request a **restriction** on how we process your data or **object** to our processing.
*   Request your data in a portable, machine-readable format (**data portability**).
*   Lodge a **complaint** with a supervisory authority. In the UK, this is the Information Commissioner's Office (ICO).

To exercise your rights, please contact us using the details below. We may need to verify your identity before processing your request.

## Children

The Service is not intended for or directed to children under the age of 13. To use this Service, you must be at least 13 years old or the minimum age required to consent to data processing in your country. By creating an account, you confirm that you meet this minimum age requirement. We do not knowingly collect personal data from children under 13. If we learn that we have inadvertently collected such data, we will take steps to delete it as quickly as possible.

## Links to other websites

The Service may allow you to run content that links to other websites not operated by us. If you follow a third-party link, you will be directed to that third party's site. We have no control over and assume no responsibility for the content, privacy policies, or practices of any third-party sites or services.

## Data breach procedures

In the event of a personal data breach, we will take immediate steps to contain and assess the impact. If the breach is likely to result in a risk to the rights and freedoms of individuals, we are prepared to notify the Information Commissioner's Office (ICO) within 72 hours. If a breach is likely to result in a high risk to your rights and freedoms, we will also notify you directly without undue delay.

## Changes to this policy

We may update this Privacy Policy from time to time. When we do, we will post the updated version on this page and revise the "Effective date" at the top. For material changes, we may provide more prominent notice, such as through the Service's dashboard or by email. This policy is reviewed regularly to ensure it remains compliant and up-to-date.

## Contact

If you have questions about this Privacy Policy or wish to exercise your rights, please contact us at: **support@hack.ngo**

## Supervisory authority

You have the right to lodge a complaint with a data protection authority. In the UK, this is the:

**Information Commissioner's Office (ICO)**
Website: https://ico.org.uk/

---
*This document is provided for transparency and does not constitute legal advice.*</pre>
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
