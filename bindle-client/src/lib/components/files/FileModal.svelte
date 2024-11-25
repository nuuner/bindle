<script lang="ts">
    import { bytesToMB } from "$lib/utils/fileUtils";
    import {
        Grid,
        Column,
        Row,
        TextInput,
        Button,
        UnorderedList,
        ListItem,
        CopyButton,
        Tile,
        ComposedModal,
        ModalHeader,
        ModalBody,
        ModalFooter,
        Truncate,
    } from "carbon-components-svelte";
    import FilePreview from "./FilePreview.svelte";
    import {
        getFileModalOpen,
        getSelectedFile,
        setFileModalOpen,
        setSelectedFile,
    } from "$lib/stores/fileStore.svelte";

    let newFileName = $state("");
    let fileNameChanged = $derived(newFileName !== getSelectedFile()?.fileName);
    let fileNameInvalid = $derived(
        (() => {
            if (!newFileName) {
                return true;
            }
            if (!newFileName.includes(".")) {
                return true;
            }
            const originalFileNameEnding = getSelectedFile()
                ?.fileName?.split(".")
                .pop();
            return !newFileName.endsWith(originalFileNameEnding ?? "");
        })(),
    );

    let file = $derived(getSelectedFile());

    $effect(() => {
        newFileName = file?.fileName ?? "";
    });
</script>

<ComposedModal
    open={getFileModalOpen()}
    on:close={() => setTimeout(() => setFileModalOpen(false), 0)}
>
    <ModalHeader label="File controls" title={file?.fileName} />
    <ModalBody>
        <div class="flex gap-4 max-h-full">
            <div class="w-1/2">
                <FilePreview {file} />
            </div>
            <div class="w-1/2 min-w-0">
                <div>
                    <TextInput
                        labelText="File name"
                        bind:value={newFileName}
                        invalid={fileNameInvalid}
                        invalidText="File name must end with the original file extension"
                    />
                </div>
                <div class="mt-2">
                    <Tile light>
                        <UnorderedList class="pl-2">
                            <ListItem>
                                {file?.mimeType}
                            </ListItem>
                            <ListItem>
                                {bytesToMB(file?.size ?? 0).toFixed(2)} MB
                            </ListItem>
                            {#if file?.details}
                                <ListItem>
                                    {file?.details}
                                </ListItem>
                            {/if}
                        </UnorderedList>
                    </Tile>
                </div>
                <div class="mt-2 flex items-center justify-between gap-2">
                    <a href={file?.url} target="_blank" class="min-w-0 flex-1">
                        <Truncate>
                            {file?.url}
                        </Truncate>
                    </a>
                    <div class="flex items-center gap-2">
                        <CopyButton
                            text={file?.url ?? ""}
                            iconDescription="Copy link"
                        />
                        <Button kind="tertiary" size="small">Reset link</Button>
                    </div>
                </div>
            </div>
        </div>
    </ModalBody>
    <ModalFooter
        primaryButtonText="Save"
        primaryButtonDisabled={!fileNameChanged || fileNameInvalid}
    />
</ComposedModal>
