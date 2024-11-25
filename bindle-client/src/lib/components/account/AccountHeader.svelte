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
        console.log("handleFileUpload", event.detail);
        const file = event.detail[0];
        uploadFile(file);
    }
</script>

<div class="flex justify-between items-center w-full">
    <div>
        <div>Current account ID</div>
        {#if getAccountId()}
            <div class="flex items-center">
                <strong class="mr-2 whitespace-nowrap">{getAccountId()}</strong>
                <CopyButton
                    text={getAccountId() ?? ""}
                    iconDescription="Copy account ID"
                />
                <OverflowMenu class="ml-2">
                    <OverflowMenuItem
                        text="Change account"
                        on:click={() => (accountChangeDialog = true)}
                    />
                    <OverflowMenuItem
                        text="Delete account"
                        danger
                        on:click={() => (deleteAccountDialog = true)}
                    />
                </OverflowMenu>
            </div>
        {:else}
            <Loading class="mt-2" small withOverlay={false} />
        {/if}
    </div>
    {#if getAccountId()}
        <div>
            <FileUploaderButton
                labelText="Upload file"
                on:change={handleFileUpload}
            />
        </div>
    {/if}
</div>
