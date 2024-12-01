<script lang="ts">
    import { InlineLoading } from "carbon-components-svelte";
    import { DocumentBlank } from "carbon-icons-svelte";
    import { bytesToMB } from "$lib/utils/fileUtils";
    import type { UploadingFile } from "$lib/stores/uploadStore.svelte";

    let { file } = $props<{ file: UploadingFile }>();
</script>

<div
    class="grid gap-4 grid-cols-[30px_minmax(0,1fr)_110px_110px] w-full opacity-70"
>
    <div class="flex items-center justify-center">
        <DocumentBlank size={20} />
    </div>
    <div class="flex items-center truncate">
        {file.fileName}
    </div>
    <div class="flex items-center">
        {bytesToMB(file.size).toFixed(2)} MB
    </div>
    <div class="flex items-center">
        <InlineLoading description="Uploading..." />
    </div>
</div>
