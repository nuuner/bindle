<script lang="ts">
	import "../app.css";

	let { children } = $props();
</script>

<div class="min-h-screen flex flex-col">
	<main class="container mx-auto mt-10 flex-grow px-4">
		{@render children()}
	</main>

	<footer class="bg-zinc-900 mt-auto">
		<div class="container mx-auto py-6 px-4">
			<div class="flex justify-between items-center">
				<div class="text-zinc-400">
					<a
						href="https://github.com/nuuner/bindle"
						class="hover:text-gray-900">Bindle</a
					>
					instance
				</div>
				<div class="flex gap-4 text-zinc-400">
					<a href="/terms" class="hover:text-zinc-200">Terms</a>
				</div>
			</div>
		</div>
	</footer>
</div>
