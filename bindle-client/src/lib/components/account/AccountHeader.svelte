<script lang="ts">
    import { uploadFile } from "$lib/services/api.svelte";
    import { getAccountId } from "$lib/stores/accountStore.client.svelte";
    import {
        CopyButton,
        OverflowMenu,
        OverflowMenuItem,
        FileUploaderButton,
        Loading,
    } from "carbon-components-svelte";

    let {
        deleteAccountDialog = $bindable(),
        accountChangeDialog = $bindable(),
    } = $props();

    function handleFileUpload(event: CustomEvent<readonly File[]>) {
        const file = event.detail[0];
        uploadFile(file);
    }
</script>

<div class="flex justify-between items-center w-full flex-wrap">
    <div>
        <div class="mb-2">Current account ID</div>
        {#if getAccountId()}
            <div class="gap-2 flex items-center flex-wrap mb-2">
                <strong class="mr-2 whitespace-nowrap">{getAccountId()}</strong>
                <div class="flex items-center flex-row">
                    <CopyButton
                        text={getAccountId() ?? ""}
                        iconDescription="Copy account ID"
                    />
                    <OverflowMenu>
                        <OverflowMenuItem
                            text="Change account"
                            on:click={() =>
                                setTimeout(() => (accountChangeDialog = true))}
                        />
                        <OverflowMenuItem
                            text="Delete account"
                            danger
                            on:click={() =>
                                setTimeout(() => (deleteAccountDialog = true))}
                        />
                    </OverflowMenu>
                </div>
            </div>
        {:else}
            <Loading class="mt-2" small withOverlay={false} />
        {/if}
    </div>
    {#if getAccountId()}
        <div>
            <FileUploaderButton
                labelText="Upload file"
                disableLabelChanges
                on:change={handleFileUpload}
            />
        </div>
    {/if}
</div>
