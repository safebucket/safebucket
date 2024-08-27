"use client"

import {Button} from "@/components/ui/button"
import {usePathname, useRouter} from "next/navigation";
import Link from "next/link";

export function SideMenu({buckets}: any) {
    const router = useRouter()
    const pathname = usePathname()
    console.log(pathname)

    return (
        <div className="w-64 border-r pr-6">
            <div className="space-y-4">
                <div>
                    <h3 className="text-lg font-medium mb-2">Private</h3>
                    <nav className="space-y-1">
                        <Link href="/"
                              className={`block py-2 px-3 rounded-md hover:bg-muted ${pathname == "/" ? "bg-muted text-primary" : ""}`}>
                            Home
                        </Link>
                        <Link href="/" className="block py-2 px-3 rounded-md hover:bg-muted">
                            Personal bucket
                        </Link>
                    </nav>
                </div>
                <div>
                    <div className="flex justify-between items-center mb-2">
                        <h3 className="text-lg font-medium">Shared buckets</h3>
                        <Button variant="outline" size="sm">
                            New
                        </Button>
                    </div>
                    <nav className="space-y-1">
                        {buckets.map((bucket: any) =>
                            <Link key={bucket.id} href={`/buckets/${bucket.id}`}
                                  className={`block py-2 px-3 rounded-md hover:bg-muted ${pathname == `/buckets/${bucket.id}` ? "bg-muted text-primary" : ""}`}>
                                {bucket.name}
                            </Link>
                        )}
                    </nav>
                </div>
                <div>
                    <h3 className="text-lg font-medium">Settings</h3>
                    <nav className="space-y-1">
                        <Link href="#" className="block py-2 px-3 rounded-md hover:bg-muted" prefetch={false}>
                            Account
                        </Link>
                        <Link href="#" className="block py-2 px-3 rounded-md hover:bg-muted" prefetch={false}>
                            Notifications
                        </Link>
                        <Link href="#" className="block py-2 px-3 rounded-md hover:bg-muted" prefetch={false}>
                            Security
                        </Link>
                    </nav>
                </div>
            </div>
        </div>
    )
}