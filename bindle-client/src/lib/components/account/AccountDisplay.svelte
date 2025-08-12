<script lang="ts">
    import {
        CopyButton,
        OverflowMenu,
        OverflowMenuItem,
        Loading,
        Button,
    } from "carbon-components-svelte";
    import QrCode from "carbon-icons-svelte/lib/QrCode.svelte";
    import { getAccountId } from "$lib/stores/accountStore.client.svelte";
    import QRCodeModal from "./QRCodeModal.svelte";

    export let onChangeAccount: () => void;
    export let onDeleteAccount: () => void;
    
    let qrCodeModalOpen = $state(false);
</script>

<div>Current account ID</div>
<div class="flex items-center">
    {#if getAccountId()}
        <strong class="mr-2 whitespace-nowrap">{getAccountId() || ""}</strong>
        <CopyButton
            text={getAccountId() || ""}
            iconDescription="Copy account ID"
        />
        <Button
            kind="ghost"
            size="small"
            icon={QrCode}
            iconDescription="Show QR Code"
            tooltipPosition="bottom"
            tooltipAlignment="center"
            on:click={() => (qrCodeModalOpen = true)}
        />
        <OverflowMenu class="ml-2">
            <OverflowMenuItem
                text="Change account"
                on:click={() => onChangeAccount()}
            />
            <OverflowMenuItem
                text="Delete account"
                danger
                on:click={() => onDeleteAccount()}
            />
        </OverflowMenu>
    {:else}
        <Loading withOverlay={false} small class="mt-2" />
    {/if}
</div>

<QRCodeModal bind:open={qrCodeModalOpen} />
