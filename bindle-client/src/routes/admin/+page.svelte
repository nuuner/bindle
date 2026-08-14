<script lang="ts">
    import { onMount } from "svelte";
    import {
        adminService,
        type AdminUser,
        type AdminFile,
        type AdminStats,
    } from "$lib/services/adminService";
    import {
        Modal,
        DataTable,
        Button,
        InlineNotification,
        PasswordInput,
        Toggle,
    } from "carbon-components-svelte";
    import StatTile from "$lib/components/admin/StatTile.svelte";
    import { formatBytes } from "$lib/utils/fileUtils";
    import TrashCan from "carbon-icons-svelte/lib/TrashCan.svelte";
    import Renew from "carbon-icons-svelte/lib/Renew.svelte";

    let password = $state("");
    let isAuthenticated = $state(false);
    let showPasswordModal = $state(true);
    let loading = $state(false);
    let error = $state("");

    let users = $state<AdminUser[]>([]);
    let files = $state<AdminFile[]>([]);
    let stats = $state<AdminStats | null>(null);

    // Most zero-file accounts are throwaways created by a visit that never uploaded
    // anything, so they are hidden by default.
    let hideUsersWithoutFiles = $state(true);

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
            [stats, users, files] = await Promise.all([
                adminService.getStats(adminPassword),
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
        { key: "accountId", value: "Account ID", width: "230px" },
        { key: "fileCount", value: "Files", width: "100px" },
        { key: "storageUsage", value: "Storage", width: "120px" },
        { key: "lastLogin", value: "Last Login", width: "180px" },
        { key: "ipAddresses", value: "IP Addresses" },
        { key: "actions", value: "Actions", width: "190px" },
    ]);

    let visibleUsers = $derived(
        hideUsersWithoutFiles ? users.filter((user) => user.fileCount > 0) : users
    );

    let hiddenUserCount = $derived(users.length - visibleUsers.length);

    let userRows = $derived(
        visibleUsers.map((user) => ({
            id: user.accountId,
            accountId: user.accountId,
            fileCount: user.fileCount,
            storageUsage: formatBytes(user.storageUsage),
            lastLogin: user.lastLogin,
            ipAddresses: user.ipAddresses.join(", "),
            actions: user.accountId,
        }))
    );

    // Explicit widths make Carbon switch the table to `table-layout: fixed`, which stops
    // one pathologically long file name from squeezing every other column into wrapping.
    // Name is left unsized so it absorbs the remaining space, and truncates.
    let fileHeaders = $derived([
        { key: "fileId", value: "File ID", width: "330px" },
        { key: "fileName", value: "Name" },
        { key: "accountId", value: "Owner", width: "230px" },
        { key: "size", value: "Size", width: "120px" },
        { key: "type", value: "Type", width: "110px" },
        { key: "createdAt", value: "Created", width: "180px" },
        { key: "actions", value: "Actions", width: "150px" },
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

        {#if stats}
            <div class="flex flex-col gap-2">
                <div class="grid grid-cols-2 md:grid-cols-4 gap-4">
                    <StatTile
                        label="Files"
                        value={stats.fileRecords.toLocaleString()}
                        hint="{stats.uniqueFiles.toLocaleString()} unique after dedup"
                    />
                    <StatTile
                        label="Stored"
                        value={formatBytes(stats.storedBytes)}
                        hint="{formatBytes(stats.logicalBytes)} logical · {formatBytes(
                            stats.dedupSavedBytes
                        )} saved"
                    />
                    <StatTile
                        label="Users with files"
                        value={stats.usersWithFiles.toLocaleString()}
                        hint="{stats.totalUsers.toLocaleString()} accounts total"
                    />
                    <StatTile
                        label="Largest file"
                        value={formatBytes(stats.largestFileBytes)}
                        hint="{formatBytes(stats.averageFileBytes)} average"
                    />
                </div>
                <p class="text-xs text-zinc-500">
                    Storage backend: {stats.storageBackend}. Stored is deduplicated content
                    size as recorded; files are encrypted at rest, so actual disk usage is
                    somewhat higher.
                </p>
            </div>
        {/if}

        <div>
            <div class="flex flex-wrap items-center justify-between gap-4 mb-4">
                <h2 class="text-2xl font-semibold">
                    Users ({visibleUsers.length}{hiddenUserCount > 0
                        ? ` of ${users.length}`
                        : ""})
                </h2>
                <Toggle
                    size="sm"
                    labelText="Hide users with no files"
                    labelA="Off"
                    labelB="On"
                    bind:toggled={hideUsersWithoutFiles}
                />
            </div>
            <div class="overflow-x-auto">
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
                            <!-- Only the unsized column can truncate, so only it gets a tooltip. -->
                            <span
                                class="block truncate"
                                title={cell.key === "ipAddresses"
                                    ? String(cell.value)
                                    : undefined}
                            >
                                {cell.value}
                            </span>
                        {/if}
                    </svelte:fragment>
                </DataTable>
            </div>
        </div>

        <div>
            <h2 class="text-2xl font-semibold mb-4">Files ({files.length})</h2>
            <div class="overflow-x-auto">
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
                            <!-- Only the unsized column can truncate, so only it gets a tooltip. -->
                            <span
                                class="block truncate"
                                title={cell.key === "fileName"
                                    ? String(cell.value)
                                    : undefined}
                            >
                                {cell.value}
                            </span>
                        {/if}
                    </svelte:fragment>
                </DataTable>
            </div>
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
    <!--
        tooltipPosition="left" keeps the visibility toggle's tooltip inside the modal's
        scrolling content area. With Carbon's default "bottom" it is absolutely positioned
        past the container's edges, which inflates scrollWidth/scrollHeight and makes the
        modal sprout stray horizontal and vertical scrollbars.
    -->
    <PasswordInput
        labelText="Admin Password"
        bind:value={password}
        placeholder="Enter admin password"
        tooltipPosition="left"
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
