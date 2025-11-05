<script lang="ts">
    import { onMount } from "svelte";
    import { adminService, type AdminUser, type AdminFile } from "$lib/services/adminService";
    import {
        Modal,
        DataTable,
        Button,
        InlineNotification,
        PasswordInput,
        Toolbar,
        ToolbarContent,
    } from "carbon-components-svelte";
    import TrashCan from "carbon-icons-svelte/lib/TrashCan.svelte";
    import Renew from "carbon-icons-svelte/lib/Renew.svelte";

    let password = $state("");
    let isAuthenticated = $state(false);
    let showPasswordModal = $state(true);
    let loading = $state(false);
    let error = $state("");

    let users = $state<AdminUser[]>([]);
    let files = $state<AdminFile[]>([]);

    let showDeleteAllModal = $state(false);
    let showDeleteUserModal = $state(false);
    let showDeleteFileModal = $state(false);
    let selectedAccountId = $state("");
    let selectedFileId = $state("");

    async function handlePasswordSubmit() {
        if (!password) {
            error = "Please enter a password";
            return;
        }

        loading = true;
        error = "";

        const isValid = await adminService.verifyPassword(password);
        if (isValid) {
            isAuthenticated = true;
            showPasswordModal = false;
            // Store password in sessionStorage for this session
            sessionStorage.setItem("adminPassword", password);
            await loadData();
        } else {
            error = "Invalid password";
        }

        loading = false;
    }

    async function loadData() {
        try {
            const adminPassword = sessionStorage.getItem("adminPassword") || password;
            [users, files] = await Promise.all([
                adminService.getAllUsers(adminPassword),
                adminService.getAllFiles(adminPassword),
            ]);
        } catch (err) {
            error = err instanceof Error ? err.message : "Failed to load data";
            // If unauthorized, clear session and show password modal again
            if (error.includes("password") || error.includes("Unauthorized")) {
                sessionStorage.removeItem("adminPassword");
                isAuthenticated = false;
                showPasswordModal = true;
            }
        }
    }

    async function handleDeleteFile(fileId: string) {
        selectedFileId = fileId;
        showDeleteFileModal = true;
    }

    async function confirmDeleteFile() {
        try {
            const adminPassword = sessionStorage.getItem("adminPassword") || password;
            await adminService.deleteFile(adminPassword, selectedFileId);
            showDeleteFileModal = false;
            await loadData();
        } catch (err) {
            error = err instanceof Error ? err.message : "Failed to delete file";
        }
    }

    async function handleDeleteUserFiles(accountId: string) {
        selectedAccountId = accountId;
        showDeleteUserModal = true;
    }

    async function confirmDeleteUserFiles() {
        try {
            const adminPassword = sessionStorage.getItem("adminPassword") || password;
            const result = await adminService.deleteUserFiles(adminPassword, selectedAccountId);
            showDeleteUserModal = false;
            error = "";
            await loadData();
        } catch (err) {
            error = err instanceof Error ? err.message : "Failed to delete user files";
        }
    }

    async function confirmDeleteAllFiles() {
        try {
            const adminPassword = sessionStorage.getItem("adminPassword") || password;
            const result = await adminService.deleteAllFiles(adminPassword);
            showDeleteAllModal = false;
            error = "";
            await loadData();
        } catch (err) {
            error = err instanceof Error ? err.message : "Failed to delete all files";
        }
    }

    function formatBytes(bytes: number): string {
        if (bytes === 0) return "0 B";
        const k = 1024;
        const sizes = ["B", "KB", "MB", "GB", "TB"];
        const i = Math.floor(Math.log(bytes) / Math.log(k));
        return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + " " + sizes[i];
    }

    onMount(() => {
        // Check if already authenticated from sessionStorage
        const storedPassword = sessionStorage.getItem("adminPassword");
        if (storedPassword) {
            password = storedPassword;
            isAuthenticated = true;
            showPasswordModal = false;
            loadData();
        }
    });

    // Prepare user table data
    let userHeaders = $derived([
        { key: "accountId", value: "Account ID" },
        { key: "fileCount", value: "Files" },
        { key: "storageUsage", value: "Storage" },
        { key: "lastLogin", value: "Last Login" },
        { key: "ipAddresses", value: "IP Addresses" },
        { key: "actions", value: "Actions" },
    ]);

    let userRows = $derived(
        users.map((user) => ({
            id: user.accountId,
            accountId: user.accountId,
            fileCount: user.fileCount,
            storageUsage: formatBytes(user.storageUsage),
            lastLogin: user.lastLogin,
            ipAddresses: user.ipAddresses.join(", "),
            actions: user.accountId,
        }))
    );

    // Prepare file table data
    let fileHeaders = $derived([
        { key: "fileId", value: "File ID" },
        { key: "fileName", value: "Name" },
        { key: "accountId", value: "Owner" },
        { key: "size", value: "Size" },
        { key: "type", value: "Type" },
        { key: "createdAt", value: "Created" },
        { key: "actions", value: "Actions" },
    ]);

    let fileRows = $derived(
        files.map((file) => ({
            id: file.fileId,
            fileId: file.fileId,
            fileName: file.fileName,
            accountId: file.accountId,
            size: formatBytes(file.size),
            type: file.type,
            createdAt: file.createdAt,
            actions: file.fileId,
        }))
    );
</script>

<svelte:head>
    <title>Admin Panel - Bindle</title>
</svelte:head>

{#if isAuthenticated}
    <div class="flex flex-col gap-6">
        <div class="flex justify-between items-center">
            <h1 class="text-3xl font-bold">Admin Panel</h1>
            <div class="flex gap-2">
                <Button
                    kind="tertiary"
                    icon={Renew}
                    on:click={loadData}
                >
                    Refresh
                </Button>
                <Button
                    kind="danger"
                    icon={TrashCan}
                    on:click={() => (showDeleteAllModal = true)}
                >
                    Delete All Files
                </Button>
            </div>
        </div>

        {#if error}
            <InlineNotification
                kind="error"
                title="Error"
                subtitle={error}
                on:close={() => (error = "")}
            />
        {/if}

        <div>
            <h2 class="text-2xl font-semibold mb-4">Users ({users.length})</h2>
            <DataTable headers={userHeaders} rows={userRows}>
                <svelte:fragment slot="cell" let:row let:cell>
                    {#if cell.key === "actions"}
                        <Button
                            size="small"
                            kind="danger-ghost"
                            icon={TrashCan}
                            on:click={() => handleDeleteUserFiles(cell.value)}
                            disabled={row.fileCount === 0}
                        >
                            Delete Files
                        </Button>
                    {:else}
                        {cell.value}
                    {/if}
                </svelte:fragment>
            </DataTable>
        </div>

        <div>
            <h2 class="text-2xl font-semibold mb-4">Files ({files.length})</h2>
            <DataTable headers={fileHeaders} rows={fileRows}>
                <svelte:fragment slot="cell" let:row let:cell>
                    {#if cell.key === "actions"}
                        <Button
                            size="small"
                            kind="danger-ghost"
                            icon={TrashCan}
                            on:click={() => handleDeleteFile(cell.value)}
                        >
                            Delete
                        </Button>
                    {:else}
                        {cell.value}
                    {/if}
                </svelte:fragment>
            </DataTable>
        </div>
    </div>
{/if}

<!-- Password Modal -->
<Modal
    bind:open={showPasswordModal}
    modalHeading="Admin Authentication"
    primaryButtonText={loading ? "Verifying..." : "Login"}
    secondaryButtonText="Cancel"
    primaryButtonDisabled={loading || !password}
    on:click:button--primary={handlePasswordSubmit}
    on:click:button--secondary={() => window.history.back()}
    preventCloseOnClickOutside
>
    <PasswordInput
        labelText="Admin Password"
        bind:value={password}
        placeholder="Enter admin password"
        on:keydown={(e) => e.key === "Enter" && handlePasswordSubmit()}
    />
    {#if error}
        <div class="mt-4">
            <InlineNotification
                kind="error"
                title="Authentication Failed"
                subtitle={error}
                hideCloseButton
            />
        </div>
    {/if}
</Modal>

<!-- Delete File Modal -->
<Modal
    bind:open={showDeleteFileModal}
    modalHeading="Delete File"
    primaryButtonText="Delete"
    secondaryButtonText="Cancel"
    on:click:button--primary={confirmDeleteFile}
    on:click:button--secondary={() => (showDeleteFileModal = false)}
    danger
>
    <p>Are you sure you want to delete this file?</p>
    <p class="text-sm text-gray-600 mt-2">File ID: {selectedFileId}</p>
</Modal>

<!-- Delete User Files Modal -->
<Modal
    bind:open={showDeleteUserModal}
    modalHeading="Delete User Files"
    primaryButtonText="Delete All"
    secondaryButtonText="Cancel"
    on:click:button--primary={confirmDeleteUserFiles}
    on:click:button--secondary={() => (showDeleteUserModal = false)}
    danger
>
    <p>Are you sure you want to delete all files for this user?</p>
    <p class="text-sm text-gray-600 mt-2">Account: {selectedAccountId}</p>
    <p class="text-sm text-red-600 mt-2"><strong>This action cannot be undone!</strong></p>
</Modal>

<!-- Delete All Files Modal -->
<Modal
    bind:open={showDeleteAllModal}
    modalHeading="Delete ALL Files"
    primaryButtonText="DELETE EVERYTHING"
    secondaryButtonText="Cancel"
    on:click:button--primary={confirmDeleteAllFiles}
    on:click:button--secondary={() => (showDeleteAllModal = false)}
    danger
>
    <p class="text-lg font-semibold">⚠️ DANGER ZONE ⚠️</p>
    <p class="mt-4">
        This will permanently delete <strong>ALL FILES</strong> from
        <strong>ALL USERS</strong> in the system!
    </p>
    <p class="text-sm text-red-600 mt-4">
        <strong>THIS ACTION CANNOT BE UNDONE!</strong>
    </p>
</Modal>
