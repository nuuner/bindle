<script lang="ts">
    import { accountService } from "$lib/services/api.svelte";
    import { getAccount } from "$lib/stores/accountStore.client.svelte";
    import { Modal, PasswordInput } from "carbon-components-svelte";

    let { open = $bindable(false) } = $props();

    let password = $state("");
    let loading = $state(false);
    let errorMessage = $state<string | undefined>(undefined);

    let unlocked = $derived(getAccount()?.limitsUnlocked ?? false);

    async function handleUnlock() {
        if (!password || loading) {
            return;
        }

        loading = true;
        errorMessage = undefined;
        try {
            await accountService.unlockLimits(password);
            password = "";
            open = false;
        } catch (error) {
            // The server answers a wrong password with 401 and no detail beyond that,
            // so there is nothing more specific to show here.
            errorMessage = "Incorrect password";
        } finally {
            loading = false;
        }
    }

    async function handleLock() {
        loading = true;
        try {
            await accountService.lockLimits();
            open = false;
        } finally {
            loading = false;
        }
    }

    function handleClose() {
        password = "";
        errorMessage = undefined;
        open = false;
    }
</script>

<Modal
    bind:open
    modalHeading="Unlock limits"
    primaryButtonText={unlocked
        ? loading
            ? "Locking..."
            : "Lock again"
        : loading
          ? "Unlocking..."
          : "Unlock"}
    secondaryButtonText="Cancel"
    primaryButtonDisabled={loading || (!unlocked && !password)}
    on:click:button--secondary={handleClose}
    on:click:button--primary={unlocked ? handleLock : handleUnlock}
    on:close={handleClose}
>
    {#if unlocked}
        <p>The daily upload limit is lifted on this browser.</p>
        <p class="mt-2">
            Lock it again to go back to the normal limit. Clearing your cookies has the
            same effect.
        </p>
    {:else}
        <p>Enter the password to remove the daily upload limit on this browser.</p>
        <div class="mt-4">
            <PasswordInput
                id="unlock-limits-password"
                labelText="Password"
                placeholder="Password"
                bind:value={password}
                invalid={!!errorMessage}
                invalidText={errorMessage}
                on:keydown={(event) => {
                    if (event.key === "Enter") {
                        handleUnlock();
                    }
                }}
            />
        </div>
    {/if}
</Modal>
