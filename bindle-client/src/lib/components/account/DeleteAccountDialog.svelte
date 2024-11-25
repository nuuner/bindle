<script lang="ts">
    import { Modal } from "carbon-components-svelte";

    let { open = $bindable(false) } = $props();

    function handleDeleteAccount() {
        // TODO: Delete account
        open = false;
    }
</script>

<Modal
    bind:open
    modalHeading="Delete account"
    primaryButtonText="Confirm"
    secondaryButtonText="Cancel"
    on:click:button--secondary={() => (open = false)}
    on:click:button--primary={handleDeleteAccount}
    danger
>
    <p>Are you sure you want to delete this account?</p>
    <p><strong>All data will be deleted!</strong></p>
</Modal>
