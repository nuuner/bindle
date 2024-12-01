<script lang="ts">
    import { ProgressBar } from "carbon-components-svelte";
    import { getAccount } from "$lib/stores/accountStore.client.svelte";
    import { bytesToMB } from "$lib/utils/fileUtils";

    // Convert bytes to MB for display
    let storageUsedInMB = $derived(bytesToMB(getAccount()?.uploadedBytes ?? 0));

    let uploadLimitMB = $derived(
        bytesToMB(getAccount()?.uploadLimitBytes ?? 0),
    );
</script>

<ProgressBar
    labelText="Upload limit"
    value={storageUsedInMB}
    max={uploadLimitMB}
    helperText={`${storageUsedInMB}MB of ${uploadLimitMB}MB per day`}
/>
