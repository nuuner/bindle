<script lang="ts">
    import { fileService } from "$lib/services/api.svelte";
    import { getAccountId } from "$lib/stores/accountStore.client.svelte";
    import {
        CopyButton,
        OverflowMenu,
        OverflowMenuItem,
        FileUploaderButton,
        Loading,
        Button,
    } from "carbon-components-svelte";
    import QrCode from "carbon-icons-svelte/lib/QrCode.svelte";
    import QRCodeModal from "./QRCodeModal.svelte";

    let {
        deleteAccountDialog = $bindable(),
        accountChangeDialog = $bindable(),
    } = $props();
    
    let qrCodeModalOpen = $state(false);

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
                <Button
                    kind="ghost"
                    size="field"
                    icon={QrCode}
                    iconDescription="Show QR Code"
                    tooltipPosition="bottom"
                    tooltipAlignment="center"
                    on:click={() => (qrCodeModalOpen = true)}
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

<QRCodeModal bind:open={qrCodeModalOpen} />
