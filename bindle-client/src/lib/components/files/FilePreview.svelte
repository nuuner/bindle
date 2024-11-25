<script lang="ts">
    import { FileType, type UploadedFile } from "$lib/types";
    import {
        CopyButton,
        ImageLoader,
        TextArea,
    } from "carbon-components-svelte";

    let { file = $bindable<UploadedFile | undefined>() } = $props();

    $effect(() => {
        if (file?.type === FileType.text && !file?.text) {
            fetch(file.url)
                .then((res) => res.text())
                .then((text) => {
                    file.text = text;
                });
        }
    });
</script>

<div
    class="w-full h-full flex items-center justify-center overflow-hidden"
    class:aspect-square={file?.type === FileType.image ||
        file?.type === FileType.video}
>
    {#if file?.type === FileType.image}
        <ImageLoader
            fadeIn
            class="max-w-full max-h-full object-contain"
            src={file?.url}
            alt={file?.fileName}
        />
    {:else if file?.type === FileType.video}
        <video
            class="max-w-full max-h-full object-contain"
            src={file?.url}
            controls
        >
            <track kind="captions" />
        </video>
    {:else if file?.type === FileType.audio}
        <audio class="w-full" src={file?.url} controls>
            <track kind="captions" />
        </audio>
    {:else if file?.type === FileType.text}
        <div class="w-full">
            <div>
                <TextArea class="w-full font-mono" value={file?.text} />
            </div>
            <div>
                <CopyButton
                    text={file?.text}
                    iconDescription="Copy text"
                    class="mt-2"
                />
            </div>
        </div>
    {/if}
</div>
