import { v4 } from "uuid";
import { browser } from '$app/environment';
import { config } from "$lib/config";
import { fetchFiles } from "./fileStore.svelte";

let accountId = $state<string | undefined>(undefined);
let ACCOUNT_ID_KEY = config.apiHost + ".bindle.accountId";

if (browser) {
    const id = localStorage.getItem(ACCOUNT_ID_KEY);
    if (id) {
        accountId = id.toUpperCase();
    } else {
        accountId = v4().toUpperCase();
        localStorage.setItem(ACCOUNT_ID_KEY, accountId);
    }
}

export const getAccountId = () => accountId;

export const setAccountId = (id: string) => {
    const upperCaseId = id.toUpperCase();
    if (browser) {
        localStorage.setItem(ACCOUNT_ID_KEY, upperCaseId);
    }
    accountId = upperCaseId;
    fetchFiles();
};