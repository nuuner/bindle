<script lang="ts">
    import { onMount } from "svelte";
    import AccountChangeDialog from "$lib/components/account/AccountChangeDialog.svelte";
    import DeleteAccountDialog from "$lib/components/account/DeleteAccountDialog.svelte";
    import FileModal from "$lib/components/files/FileModal.svelte";
    import FileList from "$lib/components/files/FileList.svelte";
    import StorageIndicator from "$lib/components/files/StorageIndicator.svelte";
    import { getFiles } from "$lib/stores/fileStore.svelte";
    import {
        getAccount,
        getAccountId,
    } from "$lib/stores/accountStore.client.svelte";
    import AccountHeader from "$lib/components/account/AccountHeader.svelte";
    import FileDropArea from "$lib/components/files/FileDropArea.svelte";
    import { getUploadingFiles } from "$lib/stores/uploadStore.svelte";
    import DashboardError from "$lib/components/errors/DashboardError.svelte";
    import { accountService } from "$lib/services/api.svelte";

    let deleteAccountDialog = $state(false);
    let accountChangeDialog = $state(false);

    onMount(() => {
        accountService.initializeAccount();
    });
</script>

<div class="flex flex-col gap-4">
    <AccountHeader bind:deleteAccountDialog bind:accountChangeDialog />
    {#if getAccountId() && getAccount()?.uploadedBytes}
        <StorageIndicator />
    {/if}
    <DashboardError />
    {#if getAccountId() && (getFiles()?.length > 0 || getUploadingFiles()?.length > 0)}
        <FileList />
    {/if}
</div>

<FileDropArea />
<DeleteAccountDialog bind:open={deleteAccountDialog} />
<AccountChangeDialog bind:open={accountChangeDialog} />
<FileModal />
