package main

import (
	"fmt"
	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/s3"
	"github.com/pulumi/pulumi-awsx/sdk/go/awsx/ec2"
	"github.com/pulumi/pulumi-eks/sdk/go/eks"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/meta/v1"
	appsv1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/apps/v1"
)


func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {

		// Create an AWS resource (S3 Bucket)
		bucket, err := s3.NewBucket(ctx, "my-bucket", nil)
		if err != nil {
			return err
		}

		// Create multiple S3 buckets
		for i := 0; i < 3; i++ {
            // Create an AWS resource (S3 Bucket)
            bucket, err := s3.NewBucket(ctx, fmt.Sprintf("my-bucket-%d", i), nil) // use a unique bucket name.
            if err != nil {
                return err
            }

            // Export the name and ID of the bucket
            ctx.Export(fmt.Sprintf("bucket%dName", i), bucket.ID())
            ctx.Export(fmt.Sprintf("bucket%dID", i), bucket.Arn)
        }

	

		imageName := "mercybassey/node-service"
		imageTag := "latest"
		imageFullName := imageName + ":" + imageTag

		// Create a VPC for our cluster.
		vpc, err := ec2.NewVpc(ctx, "vpc", nil)
		if err != nil {
			return err
		}

		
		// Create an EKS cluster
		eksCluster, err := eks.NewCluster(ctx, "my-pulumi-eks-cluster", &eks.ClusterArgs{
			Name: pulumi.String("my-pulumi-eks-cluster"),
			VpcId: vpc.VpcId,
			PublicSubnetIds:              vpc.PublicSubnetIds,
			PrivateSubnetIds:             vpc.PrivateSubnetIds,
			NodeAssociatePublicIpAddress: pulumi.BoolRef(false),
		})
		if err != nil {
			return err
		}

		eksProvider, err := kubernetes.NewProvider(ctx, "eks-provider", &kubernetes.ProviderArgs{
			Kubeconfig: eksCluster.KubeconfigJson,
		})
		if err != nil {
			return err
		}

		namespace, err := corev1.NewNamespace(ctx, "my-eks-namespace", &corev1.NamespaceArgs{
			Metadata: &metav1.ObjectMetaArgs{
				Name: pulumi.String("my-eks-namespace"),
				Labels: pulumi.StringMap{
					"name": pulumi.String("my-eks-namespace"),
				},
			},
		}, pulumi.Provider(eksProvider))
		if err != nil {
			return err
		}
		

		// Deploy the Docker image to the EKS cluster using Kubernetes deployment
		deployment, err := appsv1.NewDeployment(ctx, "my-eks-deployment", &appsv1.DeploymentArgs{
			Metadata: &metav1.ObjectMetaArgs{
				Name: pulumi.String("my-eks-deployment"),
				Namespace: pulumi.String("my-eks-namespace"),
			},
			Spec: &appsv1.DeploymentSpecArgs{
				Replicas: pulumi.Int(1),
				Selector: &metav1.LabelSelectorArgs{
					MatchLabels: pulumi.StringMap{
						"app": pulumi.String("my-app"),
					},
				},
				Template: &corev1.PodTemplateSpecArgs{
					Metadata: &metav1.ObjectMetaArgs{
						Labels: pulumi.StringMap{
							"app": pulumi.String("my-app"),
						},
					},
					Spec: &corev1.PodSpecArgs{
						Containers: corev1.ContainerArray{
							corev1.ContainerArgs{
								Image: pulumi.String(imageFullName),
								Name: pulumi.String("my-app-container"),
								Ports: corev1.ContainerPortArray{
									corev1.ContainerPortArgs{
										ContainerPort: pulumi.Int(8080),
									},
								},
							},
						},
					},
				},
			},
		}, pulumi.Provider(eksProvider))
		if err != nil {
			return err
		}

		// Export the name of the bucket
		ctx.Export("bucketName", bucket.ID())
		// Export the EKS kubeconfig
		ctx.Export("kubeconfig", eksCluster.Kubeconfig)
		ctx.Export("deployment-name", deployment.Metadata.Elem().Name())
		ctx.Export("namespace-name", namespace.Metadata.Elem().Name())

		return nil
	})
}