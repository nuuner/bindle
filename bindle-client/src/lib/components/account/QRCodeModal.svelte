<script lang="ts">
    import { Modal, Loading } from "carbon-components-svelte";
    import { onMount } from "svelte";
    import QRCode from "qrcode";
    import { getAccountId } from "$lib/stores/accountStore.client.svelte";

    let { open = $bindable(false) } = $props();
    
    let qrCanvas: HTMLCanvasElement;
    let isGenerating = $state(true);
    
    async function generateQRCode() {
        if (!qrCanvas) return;
        
        isGenerating = true;
        const accountId = getAccountId();
        if (!accountId) {
            isGenerating = false;
            return;
        }
        
        const baseUrl = window.location.origin;
        const qrUrl = `${baseUrl}?accountId=${accountId}`;
        
        try {
            await QRCode.toCanvas(qrCanvas, qrUrl, {
                width: 300,
                margin: 2,
                color: {
                    dark: '#161616',
                    light: '#ffffff'
                }
            });
        } catch (err) {
            console.error('Failed to generate QR code:', err);
        } finally {
            isGenerating = false;
        }
    }
    
    $effect(() => {
        if (open && qrCanvas) {
            generateQRCode();
        }
    });
</script>

<Modal
    bind:open
    modalHeading="Account QR Code"
    primaryButtonText="Close"
    primaryButtonIcon={undefined}
    secondaryButtonText=""
    on:click:button--primary={() => (open = false)}
    on:close={() => (open = false)}
    hasForm={false}
    size="sm"
>
    <div class="flex flex-col items-center gap-4">
        <p class="text-center text-sm">
            Scan this QR code to access this Bindle instance with your account
        </p>
        
        <div class="relative">
            {#if isGenerating}
                <div class="absolute inset-0 flex items-center justify-center bg-white">
                    <Loading small withOverlay={false} />
                </div>
            {/if}
            <canvas 
                bind:this={qrCanvas}
                class="border border-gray-300 rounded"
            ></canvas>
        </div>
        
        <p class="text-xs text-gray-300 text-center max-w-sm">
            Account ID: {getAccountId()}
        </p>
    </div>
</Modal>