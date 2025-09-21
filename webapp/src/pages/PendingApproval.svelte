<script lang="ts">
  import Header from "../lib/Header.svelte";

  export let user: {
    display_name: string;
    username: string;
    email: string;
    approval_status: string;
    rejection_reason?: string;
  };

  let verificationStatus = "none";
  let verificationUrl = "";
  let isCreatingVerification = false;
  let verificationError = "";

  async function checkVerificationStatus() {
    try {
      const response = await fetch("/user/verification/status");
      const data = await response.json();
      verificationStatus = data.status;
      verificationUrl = data.verification_url || "";
    } catch (error) {
      console.error("Failed to check verification status:", error);
    }
  }

  async function createVerificationSession() {
    isCreatingVerification = true;
    verificationError = "";

    try {
      const response = await fetch("/user/verification/create", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
      });

      const data = await response.json();

      if (response.ok) {
        verificationStatus = "not_started";
        verificationUrl = data.verification_url;
        window.location.href = data.verification_url;
      } else {
        verificationError =
          data.error || "Failed to create verification session";
      }
    } catch (error) {
      verificationError = "Network error occurred";
    } finally {
      isCreatingVerification = false;
    }
  }
  checkVerificationStatus();
</script>

<div class="min-h-screen bg-background text-foreground">
  <Header {user} currentPage="pending" />

  <main class="max-w-4xl mx-auto p-6">
    <div class="text-center py-8">
      {#if user.approval_status === "pending"}
        <div class="mb-4">
          <div
            class="w-24 h-24 mx-auto mb-4 bg-chart-3 border-2 border-border flex items-center justify-center"
          >
            <svg
              class="w-12 h-12 text-main-foreground"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                stroke-linecap="round"
                stroke-linejoin="round"
                stroke-width="2"
                d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"
              ></path>
            </svg>
          </div>
          <h1 class="text-4xl font-heading mb-4">approval pending</h1>
          <p class="text-xl text-foreground/70 max-w-2xl mx-auto mb-4">
            hey {user.display_name}! your account is waiting for admin approval
            before you can create environments.
          </p>
        </div>
        <div
          class="bg-chart-4 text-main-foreground border-2 border-border p-6 shadow-shadow max-w-2xl mx-auto mb-6"
        >
          <div class="flex items-start gap-3">
            <svg
              class="w-5 h-5 mt-0.5"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                stroke-linecap="round"
                stroke-linejoin="round"
                stroke-width="2"
                d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"
              ></path>
            </svg>
            <div class="flex-1">
              <h3 class="font-heading font-bold mb-2">
                get approved instantly!
              </h3>
              <p class="text-sm opacity-90 mb-4">
                verify your identity automatically and get approved in minutes
                instead of waiting for manual review.
              </p>

              {#if verificationStatus === "none"}
                <button
                  class="bg-background text-foreground border-2 border-border px-4 py-2 text-sm font-heading hover:translate-x-1 hover:translate-y-1 transition-transform shadow-shadow disabled:opacity-50"
                  on:click={createVerificationSession}
                  disabled={isCreatingVerification}
                >
                  {isCreatingVerification
                    ? "creating..."
                    : "verify my identity"}
                </button>
              {:else if verificationStatus === "not_started" || verificationStatus === "in_progress"}
                <div class="space-y-2">
                  <p class="text-sm opacity-90">
                    verification session created! click below to complete your
                    identity verification.
                  </p>
                  <button
                    class="bg-background text-foreground border-2 border-border px-4 py-2 text-sm font-heading hover:translate-x-1 hover:translate-y-1 transition-transform shadow-shadow"
                    on:click={() => (window.location.href = verificationUrl)}
                  >
                    continue verification
                  </button>
                </div>
              {:else if verificationStatus === "approved"}
                <p class="text-sm opacity-90">
                  ✓ identity verification completed! your account should be
                  approved shortly.
                </p>
              {/if}

              {#if verificationError}
                <p class="text-sm opacity-90 mt-2 text-chart-1">
                  error: {verificationError}
                </p>
              {/if}
            </div>
          </div>
        </div>
        <div
          class="bg-secondary-background border-2 border-border p-6 shadow-shadow max-w-2xl mx-auto mb-8"
        >
          <div class="flex items-start gap-3">
            <svg
              class="w-5 h-5 mt-0.5 text-chart-2"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                stroke-linecap="round"
                stroke-linejoin="round"
                stroke-width="2"
                d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
              ></path>
            </svg>
            <div>
              <h3 class="font-heading font-bold mb-2">
                or wait for manual approval
              </h3>
              <ul class="text-sm text-foreground/70 space-y-1 text-left">
                <li>• an admin will review your account request</li>
                <li>• you'll receive an email notification when approved</li>
                <li>
                  • once approved, you can create development environments
                </li>
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
            <div>
              <strong>status:</strong>
              <span class="text-chart-3">pending approval</span>
            </div>
          </div>
        </div>
      {:else if user.approval_status === "rejected"}
        <div class="mb-8">
          <div
            class="w-24 h-24 mx-auto mb-6 bg-chart-1 border-2 border-border flex items-center justify-center"
          >
            <svg
              class="w-12 h-12 text-main-foreground"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                stroke-linecap="round"
                stroke-linejoin="round"
                stroke-width="2"
                d="M6 18L18 6M6 6l12 12"
              ></path>
            </svg>
          </div>
          <h1 class="text-4xl font-heading mb-4">application rejected</h1>
          <p class="text-xl text-foreground/70 max-w-2xl mx-auto mb-8">
            unfortunately, your account application was not approved.
          </p>
        </div>

        {#if user.rejection_reason}
          <div
            class="bg-secondary-background border-2 border-border p-6 shadow-shadow max-w-2xl mx-auto mb-8"
          >
            <h3 class="font-heading font-bold mb-2">reason for rejection</h3>
            <p class="text-foreground/70">{user.rejection_reason}</p>
          </div>
        {/if}

        <div
          class="bg-chart-2 text-main-foreground border-2 border-border p-6 shadow-shadow max-w-2xl mx-auto"
        >
          <div class="flex items-start gap-3">
            <svg
              class="w-5 h-5 mt-0.5"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                stroke-linecap="round"
                stroke-linejoin="round"
                stroke-width="2"
                d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
              ></path>
            </svg>
            <div>
              <h4 class="font-heading font-bold mb-2">what can you do?</h4>
              <ul class="text-sm opacity-90 space-y-1 text-left">
                <li>• contact an administrator to discuss your application</li>
                <li>
                  • address any concerns mentioned in the rejection reason
                </li>
                <li>• you may be able to reapply in the future</li>
              </ul>
            </div>
          </div>
        </div>
      {/if}
    </div>
  </main>
</div>
