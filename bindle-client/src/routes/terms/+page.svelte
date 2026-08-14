<script lang="ts">
    import { config } from "$lib/config";
    import {
        Link,
        ListItem,
        OrderedList,
        UnorderedList,
    } from "carbon-components-svelte";
</script>

<div class="px-6">
    <Link href="/">Back to home</Link>

    <!-- Tailwind's preflight resets heading sizes and Carbon only styles headings via
         classes, so a bare <h1> would render at body size. -->
    <h1 class="text-3xl font-bold mt-4 mb-2">
        Terms of Service - Prohibited Content
    </h1>

    <p class="py-2">
        By using this platform, you agree to the following terms regarding
        prohibited content:
    </p>

    <OrderedList>
        <ListItem class="py-2">
            All content uploaded to this platform must comply with applicable
            local, state, federal, and international laws.
        </ListItem>
        <ListItem class="py-2"
            >Strictly prohibited content includes but is not limited to:
            <UnorderedList nested>
                <ListItem>Copyright-infringing material</ListItem>
                <ListItem>Malicious software or malware</ListItem>
                <ListItem>Illegal images, videos, or documents</ListItem>
                <ListItem
                    >Content that promotes or facilitates illegal activities</ListItem
                >
                <ListItem>Material that violates any person's rights</ListItem>
            </UnorderedList>
        </ListItem>
        <ListItem class="py-2"
            >If prohibited content is discovered or reported:
            <UnorderedList nested>
                <ListItem
                    >We will remove the content without prior notice</ListItem
                >
                <ListItem
                    >We will permanently terminate accounts associated with the
                    content</ListItem
                >
                <ListItem
                    >We will preserve and forward evidence of illegal content to
                    law enforcement agencies</ListItem
                >
                <ListItem
                    >We will cooperate fully with legal authorities in the
                    investigation of suspected illegal activities</ListItem
                >
            </UnorderedList>
        </ListItem>
        <ListItem class="py-2">
            Users who discover prohibited content should report it immediately
            to
            <Link href={`mailto:${config.contactEmail}`}
                >{config.contactEmail}</Link
            >.
        </ListItem>
        <ListItem class="py-2">
            We maintain logs of user activity that may be provided to law
            enforcement when required by law or court order.
        </ListItem>
        <ListItem class="py-2">
            Users found in violation of these terms may face criminal
            prosecution in addition to civil penalties.
        </ListItem>
    </OrderedList>
</div>
