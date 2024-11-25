<script lang="ts">
    import {
        Grid,
        Row,
        Column,
        FileUploaderButton,
    } from "carbon-components-svelte";
    import AccountChangeDialog from "$lib/components/account/AccountChangeDialog.svelte";
    import DeleteAccountDialog from "$lib/components/account/DeleteAccountDialog.svelte";
    import FileModal from "$lib/components/files/FileModal.svelte";
    import FileList from "$lib/components/files/FileList.svelte";
    import StorageIndicator from "$lib/components/files/StorageIndicator.svelte";
    import { fetchFiles, getFiles } from "$lib/stores/fileStore.svelte";
    import { getAccountId } from "$lib/stores/accountStore.client.svelte";
    import AccountDisplay from "$lib/components/account/AccountDisplay.svelte";
    import AccountHeader from "$lib/components/account/AccountHeader.svelte";

    let deleteAccountDialog = $state(false);
    let accountChangeDialog = $state(false);

    $effect(() => {
        if (getAccountId()) {
            fetchFiles();
        }
    });
</script>

<Grid padding>
    <Row>
        <AccountHeader bind:deleteAccountDialog bind:accountChangeDialog />
    </Row>
    {#if getAccountId() && getFiles().length > 0}
        <Row>
            <Column>
                <StorageIndicator />
            </Column>
        </Row>
        <Row>
            <Column>
                <FileList />
            </Column>
        </Row>
    {/if}
</Grid>

<DeleteAccountDialog bind:open={deleteAccountDialog} />
<AccountChangeDialog bind:open={accountChangeDialog} />
<FileModal />
