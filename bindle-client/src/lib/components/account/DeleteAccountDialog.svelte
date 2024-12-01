<script lang="ts">
    import { deleteAccount } from "$lib/services/api.svelte";
    import { Modal } from "carbon-components-svelte";

    let { open = $bindable(false) } = $props();

    let loading = $state(false);

    async function handleDeleteAccount() {
        loading = true;
        await deleteAccount();
        open = false;
        loading = false;
    }
</script>

<Modal
    bind:open
    modalHeading="Delete account"
    primaryButtonText={loading ? "Deleting..." : "Confirm"}
    secondaryButtonText="Cancel"
    on:click:button--secondary={() => (open = false)}
    on:click:button--primary={handleDeleteAccount}
    primaryButtonDisabled={loading}
    danger
>
    <p>Are you sure you want to delete this account?</p>
    <p><strong>All data will be deleted!</strong></p>
</Modal>
