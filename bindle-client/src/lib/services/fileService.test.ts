import { beforeEach, afterEach, describe, it, expect, vi } from 'vitest';
import { fileService, getHeaders } from './fileService';
import { setAccountId } from '$lib/stores/accountStore.client.svelte';

describe('getHeaders', () => {
    beforeEach(() => {
        setAccountId('test-account-id');
    });

    afterEach(() => {
        setAccountId(undefined);
    });

    it('should include Authorization header when account exists', () => {
        const headers = getHeaders();
        expect(headers).toEqual({
            'Content-Type': 'application/json',
            'Authorization': 'test-account-id'
        });
    });

    it('should include Authorization header with specific account id', () => {
        const headersWithSpecificAccountId = getHeaders(false, 'specific-account-id');
        expect(headersWithSpecificAccountId).toEqual({
            'Authorization': 'specific-account-id'
        });
    });

    it('should use mocked getAccountId when no specific id provided', () => {
        const headers = getHeaders(true);
        expect(headers).toEqual({
            'Content-Type': 'application/json',
            'Authorization': 'test-account-id'
        });
    });
}); 

describe('fileService.uploadFile', () => {
    it('should fail if getAccount returns falsey', async () => {
        setAccountId(undefined);
        await expect(fileService.uploadFile(new File([], 'test.txt'))).rejects.toThrowError("Account not found");
    });
});