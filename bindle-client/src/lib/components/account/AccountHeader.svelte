<script lang="ts">
    import { fileService } from "$lib/services/api.svelte";
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
        fileService.uploadFile(file);
    }
</script>

<div class="w-full">
    <div class="mb-2">Current account ID</div>
    {#if getAccountId()}
        <strong class="whitespace-nowrap">{getAccountId()}</strong>
        <div class="mt-3 flex justify-between">
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
            <div>
                <FileUploaderButton
                    size="field"
                    labelText="Upload file"
                    disableLabelChanges
                    on:change={handleFileUpload}
                />
            </div>
        </div>
    {:else}
        <Loading class="mt-2" small withOverlay={false} />
    {/if}
</div>
