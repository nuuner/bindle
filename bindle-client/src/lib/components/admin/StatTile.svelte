<script lang="ts">
    import { Tile } from "carbon-components-svelte";

    let {
        label,
        value,
        hint,
    }: { label: string; value: string | number; hint?: string } = $props();
</script>

<Tile>
    <p class="text-xs uppercase tracking-wide text-carbon-text-secondary">{label}</p>
    <p class="text-2xl font-semibold">{value}</p>
    {#if hint}
        <p class="text-xs text-carbon-text-secondary mt-1">{hint}</p>
    {/if}
</Tile>
