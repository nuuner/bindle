<script lang="ts">
    import { clearError, getError, getTimeout } from "$lib/stores/errorStore.svelte";
    import { ToastNotification } from "carbon-components-svelte";
    import { fade } from "svelte/transition";
</script>

{#if getError()}
    <div transition:fade>
        <ToastNotification
            timeout={getTimeout()}
            kind="error"
            title="Error"
            subtitle={getError() || ""}
            on:close={(e) => {
            clearError();
        }}
    />
    </div>
{/if}
