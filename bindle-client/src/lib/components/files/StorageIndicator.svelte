<script lang="ts">
    import { ProgressBar } from "carbon-components-svelte";
    import { getAccount } from "$lib/stores/accountStore.client.svelte";
    import { bytesToMB } from "$lib/utils/fileUtils";

    // Convert bytes to MB for display
    let storageUsedInMB = $derived(bytesToMB(getAccount()?.uploadedBytes ?? 0));

    let uploadLimitMB = $derived(
        bytesToMB(getAccount()?.uploadLimitBytes ?? 0),
    );

    let limitsUnlocked = $derived(getAccount()?.limitsUnlocked ?? false);
</script>

{#if limitsUnlocked}
    <!-- There is no limit to fill up, so the bar has nothing to show. The label and
         helper text keep the same shape as the bar it replaces. -->
    <div class="bx--progress-bar">
        <span class="bx--progress-bar__label">Upload limit</span>
        <div class="bx--progress-bar__helper-text">
            {storageUsedInMB}MB uploaded today, no daily limit
        </div>
    </div>
{:else}
    <ProgressBar
        labelText="Upload limit"
        value={storageUsedInMB}
        max={uploadLimitMB}
        helperText={`${storageUsedInMB}MB of ${uploadLimitMB}MB per day`}
    />
{/if}
