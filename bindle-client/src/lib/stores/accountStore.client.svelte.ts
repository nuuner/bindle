import { browser } from '$app/environment';
import type { Account } from "$lib/types";

const ACCOUNT_ID_KEY = "bindle.accountId";

let accountId = $state<string | undefined>(undefined);
let account = $state<Account | undefined>(undefined);

if (browser) {
    const id = localStorage.getItem(ACCOUNT_ID_KEY);
    if (id) {
        accountId = id;
    }
}

export const getAccountId = () => accountId;

export const setAccountId = (id: string | undefined) => {
    if (browser && id) {
        localStorage.setItem(ACCOUNT_ID_KEY, id);
    } else if (browser) {
        localStorage.removeItem(ACCOUNT_ID_KEY);
    }
    accountId = id;
};

export const getAccount = () => account;

export const setAccount = (acc: Account) => {
    account = acc;
};
