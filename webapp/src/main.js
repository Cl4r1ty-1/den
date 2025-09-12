import { createInertiaApp } from '@inertiajs/svelte'
import './app.css'

createInertiaApp({
	resolve: name => import(`./pages/${name}.svelte`),
	setup({ el, App, props }) {
		new App({ target: el, props })
	},
	progress: { showSpinner: true }
})
