<script lang="ts">
    import { setAccountId } from "$lib/stores/accountStore.client.svelte";
    import { Modal, TextInput } from "carbon-components-svelte";

    let { open = $bindable(false) } = $props();

    let changeAccountId = $state("");
    let changeAccountIdValid = $derived(changeAccountId.length === 36);

    function handleChangeAccount() {
        if (!changeAccountIdValid) {
            return;
        }

        setAccountId(changeAccountId);
        open = false;
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
            invalid={!changeAccountIdValid}
            invalidText="Invalid account ID"
        />
    </div>
</Modal>
