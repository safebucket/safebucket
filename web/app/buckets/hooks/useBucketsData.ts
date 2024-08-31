import useSWR from 'swr'
import {fetcher} from "@/app/helpers/utils";
import {IBucketsData} from "@/app/buckets/helpers/types";

export const useBucketsData = (): IBucketsData => {
    const {data, error, isLoading} = useSWR(`/buckets`, fetcher)

    return {
        buckets: data?.data,
        error,
        isLoading,
    };
};
