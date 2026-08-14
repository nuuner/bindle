<script lang="ts">
	import "../app.css";
	import { Link } from "carbon-components-svelte";

	let { children } = $props();
</script>

<div class="w-screen h-screen flex flex-col">
	<main class="container mx-auto mt-10 flex-grow px-4">
		{@render children()}
	</main>

	<footer class="bg-carbon-layer mt-auto">
		<div class="container mx-auto py-6 px-4">
			<div class="flex justify-between items-center">
				<div class="text-carbon-text-secondary">
					<Link href="https://github.com/nuuner/bindle">Bindle</Link>
					instance
				</div>
				<div class="flex gap-4">
					<Link href="/terms">Terms</Link>
				</div>
			</div>
		</div>
	</footer>
</div>
