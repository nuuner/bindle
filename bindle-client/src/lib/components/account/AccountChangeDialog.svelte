<script lang="ts">
    import { setAccountId } from "$lib/stores/accountStore.client.svelte";
    import { Modal, TextInput } from "carbon-components-svelte";

    let { open = $bindable(false) } = $props();

    let changeAccountId = $state("");
    let changeAccountIdValid = $derived(changeAccountId.length === 36);
    let errorMessage = $state<string | undefined>(undefined);

    async function handleChangeAccount() {
        if (!changeAccountIdValid) {
            return;
        }

        try {
            await setAccountId(changeAccountId);
            open = false;
        } catch (error) {
            changeAccountId = "";
            errorMessage = "Cannot login with this account ID";
        }
    }
</script>

<Modal
    bind:open
    modalHeading="Change account"
    primaryButtonText="Change"
    secondaryButtonText="Cancel"
    on:click:button--secondary={() => (open = false)}
    on:click:button--primary={handleChangeAccount}
>
    <p>Enter the account ID you want to change to</p>
    <div class="mt-4">
        <TextInput
            id="change-account-id"
            labelText="Account ID"
            bind:value={changeAccountId}
            invalid={!changeAccountIdValid || !!errorMessage}
            invalidText={errorMessage || "Invalid account ID"}
        />
    </div>
</Modal>
