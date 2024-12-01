import { v4 } from "uuid";
import { browser } from '$app/environment';
import { config } from "$lib/config";
import { refreshMe } from "./fileStore.svelte";
import type { Account } from "$lib/types";

let accountId = $state<string | undefined>(undefined);
let ACCOUNT_ID_KEY = config.apiHost + ".bindle.accountId";

let account = $state<Account | undefined>(undefined);

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
    refreshMe();
};

export const getAccount = () => account;
export const setAccount = (acc: Account) => {
    account = acc;
};
