<script lang="ts">
    import { Column, ProgressBar, Row } from "carbon-components-svelte";
    import { getFiles } from "$lib/stores/fileStore.svelte";
    import { bytesToMB } from "$lib/utils/fileUtils";
    let storageUsedInMB = $derived(
        bytesToMB(getFiles().reduce((acc, file) => acc + file.size, 0)),
    );
</script>

<ProgressBar
    labelText="Storage used"
    value={storageUsedInMB}
    max={1000}
    helperText={`${storageUsedInMB}MB of 1GB`}
/>
