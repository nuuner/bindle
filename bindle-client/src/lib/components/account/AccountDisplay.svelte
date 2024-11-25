<script lang="ts">
    import {
        CopyButton,
        OverflowMenu,
        OverflowMenuItem,
        Loading,
    } from "carbon-components-svelte";
    import { getAccountId } from "$lib/stores/accountStore.client.svelte";

    export let onChangeAccount: () => void;
    export let onDeleteAccount: () => void;
</script>

<div>Current account ID</div>
<div class="flex items-center">
    {#if getAccountId()}
        <strong class="mr-2 whitespace-nowrap">{getAccountId() || ""}</strong>
        <CopyButton
            text={getAccountId() || ""}
            iconDescription="Copy account ID"
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
