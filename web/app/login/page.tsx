import Link from "next/link";
import {Card, CardContent, CardFooter} from "@/components/ui/card";
import {Button} from "@/components/ui/button";
import {Label} from "@/components/ui/label";
import {Input} from "@/components/ui/input";
import {Chrome, Github} from "lucide-react";

export default function Login() {
    return (
        <div className="flex-1 m-6">
            <div className="grid grid-cols-1 gap-8">
                <div className="items-center justify-center px-4 py-12 sm:px-6 lg:px-8">
                    <div className="mx-auto max-w-md space-y-4">
                        <div className="text-center">
                            <h1 className="text-3xl font-bold tracking-tight text-foreground">Sign in to your
                                account</h1>
                            <p className="mt-2 text-muted-foreground">
                                Don&apos;t have an account?{" "}
                                <Link href="#" className="font-medium text-primary hover:underline" prefetch={false}>
                                    Register
                                </Link>
                            </p>
                        </div>
                        <Card>
                            <CardContent className="space-y-4">
                                <div className="grid grid-cols-2 gap-4 mt-4">
                                    <Button variant="outline">
                                        <Chrome className="mr-2 h-4 w-4"/>
                                        Google
                                    </Button>
                                    <Button variant="outline">
                                        <Github className="mr-2 h-4 w-4"/>
                                        Github
                                    </Button>
                                </div>
                                <div className="relative">
                                    <div className="absolute inset-0 flex items-center">
                                        <span className="w-full border-t"/>
                                    </div>
                                    <div className="relative flex justify-center text-xs uppercase">
                                        <span
                                            className="bg-background px-2 text-muted-foreground">Or continue with</span>
                                    </div>
                                </div>
                                <div className="grid gap-2">
                                    <Label htmlFor="email">Email</Label>
                                    <Input id="email" type="email" placeholder="name@example.com" required/>
                                </div>
                                <div className="grid gap-2">
                                    <div className="flex items-center justify-between">
                                        <Label htmlFor="password">Password</Label>
                                        <Link href="#" className="text-sm font-medium text-primary hover:underline"
                                              prefetch={false}>
                                            Forgot password?
                                        </Link>
                                    </div>
                                    <Input id="password" type="password" required/>
                                </div>
                            </CardContent>
                            <CardFooter>
                                <Button type="submit" className="w-full">
                                    Sign in
                                </Button>
                            </CardFooter>
                        </Card>
                    </div>
                </div>
            </div>
        </div>
    )
}